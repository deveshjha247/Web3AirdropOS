package services

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/cryptoautomation/backend/internal/models"
	"github.com/cryptoautomation/backend/internal/websocket"
)

type TaskService struct {
	container *Container
}

func NewTaskService(c *Container) *TaskService {
	return &TaskService{container: c}
}

type UpdateTaskRequest struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	TargetURL      string  `json:"target_url"`
	RequiredAction string  `json:"required_action"`
	IsAutomatable  *bool   `json:"is_automatable"`
	RequiresManual *bool   `json:"requires_manual"`
	Points         *int    `json:"points"`
	Order          *int    `json:"order"`
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
	task, err := s.Get(userID, taskID)
	if err != nil {
		return nil, err
	}

	// Check dependencies
	if task.DependsOn != nil && !req.Force {
		var depExecution models.TaskExecution
		err := s.container.DB.Where("task_id = ? AND status = ?", task.DependsOn, "completed").First(&depExecution).Error
		if err != nil {
			return nil, errors.New("dependency task not completed yet")
		}
	}

	// Create execution record
	execution := &models.TaskExecution{
		ID:        uuid.New(),
		TaskID:    taskID,
		WalletID:  req.WalletID,
		AccountID: req.AccountID,
		Status:    "in_progress",
		StartedAt: time.Now(),
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
	err = s.executeTaskByType(userID, task, execution)
	if err != nil {
		execution.Status = "failed"
		execution.ErrorMessage = err.Error()
		s.container.DB.Save(execution)

		s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
			Level:   "error",
			Source:  "task",
			Message: "❌ Task failed: " + err.Error(),
			TaskID:  taskID.String(),
		})

		return execution, err
	}

	now := time.Now()
	execution.Status = "completed"
	execution.CompletedAt = &now
	s.container.DB.Save(execution)

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "success",
		Source:  "task",
		Message: "✅ Task completed: " + task.Name,
		TaskID:  taskID.String(),
	})

	s.container.WSHub.BroadcastTaskUpdate(userID.String(), websocket.TaskStatusUpdate{
		TaskID:  taskID.String(),
		Status:  "completed",
		Message: "Task completed successfully",
	})

	return execution, nil
}

func (s *TaskService) executeTaskByType(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	switch task.Type {
	case models.TaskTypeConnect:
		return s.executeWalletConnect(userID, task, execution)
	case models.TaskTypeTransaction:
		return s.executeTransaction(userID, task, execution)
	case models.TaskTypeClaim:
		return s.executeClaim(userID, task, execution)
	case models.TaskTypeFollow:
		return s.executeFollow(userID, task, execution)
	case models.TaskTypeJoin:
		return s.executeJoin(userID, task, execution)
	case models.TaskTypePost:
		return s.executePost(userID, task, execution)
	case models.TaskTypeReply:
		return s.executeReply(userID, task, execution)
	case models.TaskTypeLike:
		return s.executeLike(userID, task, execution)
	case models.TaskTypeRecast:
		return s.executeRecast(userID, task, execution)
	case models.TaskTypeVerify:
		return s.executeVerify(userID, task, execution)
	default:
		return errors.New("unsupported task type")
	}
}

func (s *TaskService) executeWalletConnect(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// This typically requires browser automation - mark as waiting for manual
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Wallet connect task - opening browser session...",
		TaskID:  task.ID.String(),
	})
	// TODO: Start browser session and navigate to target URL
	return nil
}

func (s *TaskService) executeTransaction(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Prepare transaction - actual signing happens in browser
	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Preparing transaction...",
		TaskID:  task.ID.String(),
	})
	// TODO: Prepare unsigned transaction and present for signing
	return nil
}

func (s *TaskService) executeClaim(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Claim task - often requires browser
	return nil
}

func (s *TaskService) executeFollow(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Follow a user on platform
	if execution.AccountID == nil {
		return errors.New("account required for follow task")
	}

	var account models.PlatformAccount
	if err := s.container.DB.First(&account, execution.AccountID).Error; err != nil {
		return err
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:     "info",
		Source:    "task",
		Message:   "Following " + task.TargetAccount + " on " + task.TargetPlatform,
		TaskID:    task.ID.String(),
		AccountID: account.ID.String(),
	})

	// TODO: Execute follow via platform API or browser
	return nil
}

func (s *TaskService) executeJoin(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Join a group/community
	return nil
}

func (s *TaskService) executePost(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Create a post
	if execution.AccountID == nil {
		return errors.New("account required for post task")
	}

	s.container.WSHub.BroadcastTerminal(userID.String(), websocket.TerminalMessage{
		Level:   "info",
		Source:  "task",
		Message: "Creating post on " + task.TargetPlatform,
		TaskID:  task.ID.String(),
	})

	// TODO: Create post via platform API
	return nil
}

func (s *TaskService) executeReply(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Reply to a post
	return nil
}

func (s *TaskService) executeLike(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Like a post
	return nil
}

func (s *TaskService) executeRecast(userID uuid.UUID, task *models.CampaignTask, execution *models.TaskExecution) error {
	// Recast/retweet a post
	return nil
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
