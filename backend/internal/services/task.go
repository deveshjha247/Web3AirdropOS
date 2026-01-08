package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/web3airdropos/backend/internal/models"
	"github.com/web3airdropos/backend/internal/services/platforms"
	"github.com/web3airdropos/backend/internal/websocket"
)

type TaskService struct {
	container   *Container
	adapters    map[string]platforms.PlatformAdapter
	rateLimiter *RateLimiter
	audit       *AuditService
}

func NewTaskService(c *Container) *TaskService {
	return &TaskService{
		container:   c,
		adapters:    make(map[string]platforms.PlatformAdapter),
		rateLimiter: c.RateLimiter,
		audit:       c.Audit,
	}
}

// RegisterAdapter registers a platform adapter
func (s *TaskService) RegisterAdapter(platform string, adapter platforms.PlatformAdapter) {
	s.adapters[platform] = adapter
}

// GetAdapter returns the appropriate adapter for a platform
func (s *TaskService) GetAdapter(platform string) (platforms.PlatformAdapter, error) {
	if adapter, ok := s.adapters[platform]; ok {
		return adapter, nil
	}
	return nil, fmt.Errorf("no adapter registered for platform: %s", platform)
}

type UpdateTaskRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	TargetURL      string `json:"target_url"`
	RequiredAction string `json:"required_action"`
	IsAutomatable  *bool  `json:"is_automatable"`
	RequiresManual *bool  `json:"requires_manual"`
	Points         *int   `json:"points"`
	Order          *int   `json:"order"`
}

type ExecuteTaskRequest struct {
	WalletID  *uuid.UUID `json:"wallet_id"`
	AccountID *uuid.UUID `json:"account_id"`
	Force     bool       `json:"force"` // Force execute even if dependencies not met
}

func (s *TaskService) Get(userID, taskID uuid.UUID) (*models.CampaignTask, error) {
	var task models.CampaignTask
	if err := s.container.DB.
		Preload("Executions").
		First(&task, taskID).Error; err != nil {
		return nil, err
	}

	// Verify ownership through campaign
	var campaign models.Campaign
	if err := s.container.DB.Where("id = ? AND user_id = ?", task.CampaignID, userID).First(&campaign).Error; err != nil {
		return nil, errors.New("task not found")
	}

	return &task, nil
}

func (s *TaskService) Update(userID, taskID uuid.UUID, req *UpdateTaskRequest) (*models.CampaignTask, error) {
	task, err := s.Get(userID, taskID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.TargetURL != "" {
		updates["target_url"] = req.TargetURL
	}
	if req.RequiredAction != "" {
		updates["required_action"] = req.RequiredAction
	}
	if req.IsAutomatable != nil {
		updates["is_automatable"] = *req.IsAutomatable
	}
	if req.RequiresManual != nil {
		updates["requires_manual"] = *req.RequiresManual
	}
	if req.Points != nil {
		updates["points"] = *req.Points
	}
	if req.Order != nil {
		updates["order"] = *req.Order
	}

	if err := s.container.DB.Model(task).Updates(updates).Error; err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) Execute(userID, taskID uuid.UUID, req *ExecuteTaskRequest) (*models.TaskExecution, error) {
	ctx := context.Background()

	task, err := s.Get(userID, taskID)
	if err != nil {
		return nil, err
	}

	// Generate idempotency key
	idempotencyKey := s.generateIdempotencyKey(userID, taskID, req)

	// Check for existing execution with same idempotency key
	var existingExecution models.TaskExecution
	if err := s.container.DB.Where("idempotency_key = ?", idempotencyKey).First(&existingExecution).Error; err == nil {
		// Already executed
		s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
			Level:   "warn",
			Source:  "task",
			Message: "⚠️ Task already executed (idempotency check)",
			TaskID:  taskID.String(),
		})
		return &existingExecution, nil
	}

	// Check dependencies
	if task.DependsOn != nil && !req.Force {
		var depExecution models.TaskExecution
		err := s.container.DB.Where("task_id = ? AND status = ?", task.DependsOn, "completed").First(&depExecution).Error
		if err != nil {
			return nil, errors.New("dependency task not completed yet")
		}
	}

	// Acquire rate limit slot (if applicable)
	if req.AccountID != nil && task.TargetPlatform != "" {
		allowed, err := s.rateLimiter.CheckRateLimit(ctx, task.TargetPlatform, req.AccountID.String())
		if err != nil {
			return nil, fmt.Errorf("rate limit check failed: %w", err)
		}
		if !allowed {
			return nil, errors.New("rate limit exceeded for this platform")
		}
	}

	// Create execution record
	execution := &models.TaskExecution{
		ID:             uuid.New(),
		TaskID:         taskID,
		WalletID:       req.WalletID,
		AccountID:      req.AccountID,
		Status:         "in_progress",
		IdempotencyKey: idempotencyKey,
		StartedAt:      time.Now(),
	}

	if err := s.container.DB.Create(execution).Error; err != nil {
		return nil, err
	}

	// Broadcast terminal message
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Starting task: " + task.Name,
		TaskID:  taskID.String(),
		Details: map[string]interface{}{
			"type":            task.Type,
			"target_url":      task.TargetURL,
			"requires_manual": task.RequiresManual,
			"idempotency_key": idempotencyKey,
		},
	})

	// Check if requires manual intervention
	if task.RequiresManual {
		execution.Status = "waiting_manual"
		s.container.DB.Save(execution)

		s.container.WSHub.BroadcastTaskUpdate(userID.String(), websocket.TaskStatusUpdate{
			TaskID:         taskID.String(),
			Status:         "waiting_manual",
			Message:        task.RequiredAction,
			RequiresManual: true,
		})

		s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
			Level:   "warn",
			Source:  "task",
			Message: "⚠️ Manual action required: " + task.RequiredAction,
			TaskID:  taskID.String(),
		})

		return execution, nil
	}

	// Execute based on task type
	proof, err := s.executeTaskByType(ctx, userID, task, execution)

	// Record rate limit action on success
	if err == nil && req.AccountID != nil && task.TargetPlatform != "" {
		s.rateLimiter.RecordAction(ctx, task.TargetPlatform, req.AccountID.String())
	}

	if err != nil {
		execution.Status = "failed"
		execution.ErrorMessage = err.Error()
		s.container.DB.Save(execution)

		// Log failure to audit
		if s.audit != nil {
			s.audit.LogTaskExecution(ctx, execution, task, models.ResultFailed, nil, err)
		}

		s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
			Level:   "error",
			Source:  "task",
			Message: "❌ Task failed: " + err.Error(),
			TaskID:  taskID.String(),
		})

		return execution, err
	}

	// Store proof
	if proof != nil {
		execution.ProofType = getProofTypeFromAdapter(proof)
		execution.ProofValue = getProofValueFromAdapter(proof)
		execution.PostID = proof.PostID
		execution.PostURL = proof.PostURL
	}

	now := time.Now()
	execution.Status = "completed"
	execution.CompletedAt = &now
	s.container.DB.Save(execution)

	// Log success to audit
	if s.audit != nil {
		s.audit.LogTaskExecution(ctx, execution, task, models.ResultSuccess, proof, nil)
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "success",
		Source:  "task",
		Message: "✅ Task completed: " + task.Name,
		TaskID:  taskID.String(),
		Details: map[string]interface{}{
			"proof_type":  execution.ProofType,
			"proof_value": execution.ProofValue,
		},
	})

	s.container.WSHub.BroadcastTaskUpdate(userID.String(), websocket.TaskStatusUpdate{
		TaskID:  taskID.String(),
		Status:  "completed",
		Message: "Task completed successfully",
	})

	return execution, nil
}

// generateIdempotencyKey creates a unique key for a task execution
func (s *TaskService) generateIdempotencyKey(userID, taskID uuid.UUID, req *ExecuteTaskRequest) string {
	data := fmt.Sprintf("%s:%s:", userID.String(), taskID.String())
	if req.AccountID != nil {
		data += req.AccountID.String()
	}
	if req.WalletID != nil {
		data += req.WalletID.String()
	}
	// Add date to allow re-execution on different days for daily tasks
	data += time.Now().Format("2006-01-02")

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *TaskService) executeTaskByType(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) (*platforms.ActionProof, error) {
	switch task.Type {
	case models.TaskTypeConnect:
		return nil, s.executeWalletConnect(userID, task, execution)
	case models.TaskTypeTransaction:
		return nil, s.executeTransaction(userID, task, execution)
	case models.TaskTypeClaim:
		return nil, s.executeClaim(userID, task, execution)
	case models.TaskTypeFollow:
		return s.executeFollowWithAdapter(ctx, userID, task, execution)
	case models.TaskTypeJoin:
		return nil, s.executeJoin(userID, task, execution)
	case models.TaskTypePost:
		return s.executePostWithAdapter(ctx, userID, task, execution)
	case models.TaskTypeReply:
		return s.executeReplyWithAdapter(ctx, userID, task, execution)
	case models.TaskTypeLike:
		return s.executeLikeWithAdapter(ctx, userID, task, execution)
	case models.TaskTypeRecast:
		return s.executeRecastWithAdapter(ctx, userID, task, execution)
	case models.TaskTypeVerify:
		return nil, s.executeVerify(userID, task, execution)
	default:
		return nil, errors.New("unsupported task type")
	}
}

func (s *TaskService) executeWalletConnect(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Wallet connect requires browser automation - signal the browser service
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Wallet connect task - initiating browser session...",
		TaskID:  task.ID.String(),
	})

	// Update execution to pending - requires user interaction in browser
	execution.Status = "pending"
	execution.ErrorMessage = "Awaiting wallet connection in browser"
	s.container.DB.Save(execution)

	// Broadcast browser action request
	s.container.WSHub.BroadcastToUser(userID.String(), "browser:action", map[string]interface{}{
		"action":       "wallet_connect",
		"task_id":      task.ID.String(),
		"execution_id": execution.ID.String(),
		"target_url":   task.TargetURL,
	})

	return nil
}

func (s *TaskService) executeTransaction(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Transaction execution requires browser wallet interaction
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Preparing transaction for signing...",
		TaskID:  task.ID.String(),
	})

	// Parse transaction details from task config
	var txConfig struct {
		ContractAddress string `json:"contract_address"`
		ChainID         int    `json:"chain_id"`
		FunctionName    string `json:"function_name"`
		Value           string `json:"value"`
	}
	if task.Config != "" {
		// Config contains transaction details
		_ = txConfig // Config parsing happens on frontend
	}

	// Update execution to pending - requires signature
	execution.Status = "pending"
	execution.ErrorMessage = "Awaiting transaction signature in browser"
	s.container.DB.Save(execution)

	// Broadcast transaction request to user's browser
	s.container.WSHub.BroadcastToUser(userID.String(), "browser:action", map[string]interface{}{
		"action":       "sign_transaction",
		"task_id":      task.ID.String(),
		"execution_id": execution.ID.String(),
		"target_url":   task.TargetURL,
		"tx_config":    task.Config,
	})

	return nil
}

func (s *TaskService) executeClaim(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Claim task - often requires browser
	return nil
}

func (s *TaskService) executeFollow(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Legacy - use executeFollowWithAdapter instead
	return errors.New("use executeFollowWithAdapter")
}

// executeFollowWithAdapter executes a follow using the platform adapter
func (s *TaskService) executeFollowWithAdapter(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) (*platforms.ActionProof, error) {
	if execution.AccountID == nil {
		return nil, errors.New("account required for follow task")
	}

	// Get the platform account
	var account models.PlatformAccount
	if err := s.container.DB.First(&account, execution.AccountID).Error; err != nil {
		return nil, err
	}

	// Get adapter
	adapter, err := s.GetAdapter(task.TargetPlatform)
	if err != nil {
		// Fallback: log and return nil (manual execution needed)
		s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
			Level:   "warn",
			Source:  "task",
			Message: fmt.Sprintf("No adapter for %s, requires manual execution", task.TargetPlatform),
			TaskID:  task.ID.String(),
		})
		return nil, nil
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:     "info",
		Source:    "task",
		Message:   "Following " + task.TargetAccount + " on " + task.TargetPlatform,
		TaskID:    task.ID.String(),
		AccountID: account.ID.String(),
	})

	// Acquire account lock (one action at a time per account)
	lock, err := s.rateLimiter.AccountLock(ctx, *execution.AccountID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not acquire account lock: %w", err)
	}
	defer lock.Release(ctx)

	// Execute follow via adapter
	return adapter.Follow(ctx, task.TargetAccount)
}

func (s *TaskService) executeJoin(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Join a group/community - typically requires browser
	return nil
}

func (s *TaskService) executePost(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Legacy - use executePostWithAdapter
	return errors.New("use executePostWithAdapter")
}

// executePostWithAdapter creates a post using the platform adapter
func (s *TaskService) executePostWithAdapter(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) (*platforms.ActionProof, error) {
	if execution.AccountID == nil {
		return nil, errors.New("account required for post task")
	}

	// Get content from task config or content draft
	var content string
	if task.Config != "" {
		// Try to parse content from task config
		var cfg struct {
			Content        string `json:"content"`
			ContentDraftID string `json:"content_draft_id"`
		}
		if err := json.Unmarshal([]byte(task.Config), &cfg); err == nil {
			if cfg.Content != "" {
				content = cfg.Content
			} else if cfg.ContentDraftID != "" {
				// Fetch from content drafts table
				var draft models.ContentDraft
				if err := s.container.DB.First(&draft, "id = ?", cfg.ContentDraftID).Error; err == nil {
					content = draft.Content
				}
			}
		}
	}

	// Fall back to required action if no content in config
	if content == "" {
		content = task.RequiredAction
	}

	if content == "" {
		return nil, errors.New("no content specified for post")
	}

	// Get adapter
	adapter, err := s.GetAdapter(task.TargetPlatform)
	if err != nil {
		return nil, nil // Manual execution needed
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Creating post on " + task.TargetPlatform,
		TaskID:  task.ID.String(),
	})

	// Acquire account lock
	lock, err := s.rateLimiter.AccountLock(ctx, *execution.AccountID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not acquire account lock: %w", err)
	}
	defer lock.Release(ctx)

	// Execute via adapter
	return adapter.Post(ctx, &platforms.PostContent{Text: content})
}

func (s *TaskService) executeReply(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	return errors.New("use executeReplyWithAdapter")
}

// executeReplyWithAdapter replies to a post using the platform adapter
func (s *TaskService) executeReplyWithAdapter(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) (*platforms.ActionProof, error) {
	if execution.AccountID == nil {
		return nil, errors.New("account required for reply task")
	}

	content := task.RequiredAction
	if content == "" {
		return nil, errors.New("no content specified for reply")
	}

	adapter, err := s.GetAdapter(task.TargetPlatform)
	if err != nil {
		return nil, nil
	}

	// Acquire account lock
	lock, err := s.rateLimiter.AccountLock(ctx, *execution.AccountID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not acquire account lock: %w", err)
	}
	defer lock.Release(ctx)

	// TargetID is the post ID to reply to
	return adapter.Reply(ctx, task.TargetURL, &platforms.PostContent{Text: content})
}

func (s *TaskService) executeLike(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	return errors.New("use executeLikeWithAdapter")
}

// executeLikeWithAdapter likes a post using the platform adapter
func (s *TaskService) executeLikeWithAdapter(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) (*platforms.ActionProof, error) {
	if execution.AccountID == nil {
		return nil, errors.New("account required for like task")
	}

	adapter, err := s.GetAdapter(task.TargetPlatform)
	if err != nil {
		return nil, nil
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Liking post on " + task.TargetPlatform,
		TaskID:  task.ID.String(),
	})

	// Acquire account lock
	lock, err := s.rateLimiter.AccountLock(ctx, *execution.AccountID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not acquire account lock: %w", err)
	}
	defer lock.Release(ctx)

	return adapter.Like(ctx, task.TargetURL)
}

func (s *TaskService) executeRecast(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	return errors.New("use executeRecastWithAdapter")
}

// executeRecastWithAdapter reposts/recasts using the platform adapter
func (s *TaskService) executeRecastWithAdapter(ctx context.Context, userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) (*platforms.ActionProof, error) {
	if execution.AccountID == nil {
		return nil, errors.New("account required for recast task")
	}

	adapter, err := s.GetAdapter(task.TargetPlatform)
	if err != nil {
		return nil, nil
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Recasting post on " + task.TargetPlatform,
		TaskID:  task.ID.String(),
	})

	// Acquire account lock
	lock, err := s.rateLimiter.AccountLock(ctx, *execution.AccountID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not acquire account lock: %w", err)
	}
	defer lock.Release(ctx)

	return adapter.Repost(ctx, task.TargetURL)
}

func (s *TaskService) executeVerify(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Verify task completion
	return nil
}

// Continue resumes a task that was waiting for manual action
func (s *TaskService) Continue(userID, taskID, executionID uuid.UUID, result map[string]interface{}) error {
	// Verify ownership
	_, err := s.Get(userID, taskID)
	if err != nil {
		return err
	}

	var execution models.TaskExecution
	if err := s.container.DB.Where("id = ? AND task_id = ?", executionID, taskID).First(&execution).Error; err != nil {
		return err
	}

	if execution.Status != "waiting_manual" {
		return errors.New("task is not waiting for manual action")
	}

	// Update execution with result
	now := time.Now()
	execution.Status = "completed"
	execution.CompletedAt = &now

	if txHash, ok := result["transaction_hash"].(string); ok {
		execution.TransactionHash = txHash
	}

	if err := s.container.DB.Save(&execution).Error; err != nil {
		return err
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "success",
		Source:  "task",
		Message: "✅ Manual task completed",
		TaskID:  taskID.String(),
	})

	s.container.WSHub.BroadcastTaskUpdate(userID.String(), websocket.TaskStatusUpdate{
		TaskID:  taskID.String(),
		Status:  "completed",
		Message: "Manual action completed",
	})

	return nil
}

func (s *TaskService) GetExecutions(userID, taskID uuid.UUID) ([]models.TaskExecution, error) {
	_, err := s.Get(userID, taskID)
	if err != nil {
		return nil, err
	}

	var executions []models.TaskExecution
	if err := s.container.DB.Where("task_id = ?", taskID).Order("created_at DESC").Find(&executions).Error; err != nil {
		return nil, err
	}

	return executions, nil
}

// Helper functions
func getProofTypeFromAdapter(proof *platforms.ActionProof) string {
	if proof.TxHash != "" {
		return "tx_hash"
	}
	if proof.CastHash != "" {
		return "cast_hash"
	}
	if proof.PostURL != "" {
		return "post_url"
	}
	if proof.ScreenshotPath != "" {
		return "screenshot"
	}
	if proof.PostID != "" {
		return "post_id"
	}
	return ""
}

func getProofValueFromAdapter(proof *platforms.ActionProof) string {
	if proof.TxHash != "" {
		return proof.TxHash
	}
	if proof.CastHash != "" {
		return proof.CastHash
	}
	if proof.PostURL != "" {
		return proof.PostURL
	}
	if proof.ScreenshotPath != "" {
		return proof.ScreenshotPath
	}
	if proof.PostID != "" {
		return proof.PostID
	}
	return ""
}
