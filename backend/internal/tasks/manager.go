package tasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/locks"
	"github.com/web3airdropos/backend/internal/queue"
)

// TaskStatus represents the status of a task execution
type TaskStatus string

const (
	StatusPending        TaskStatus = "PENDING"
	StatusRunning        TaskStatus = "RUNNING"
	StatusManualRequired TaskStatus = "MANUAL_REQUIRED"
	StatusDone           TaskStatus = "DONE"
	StatusFailed         TaskStatus = "FAILED"
	StatusSkipped        TaskStatus = "SKIPPED"
)

// ProofType represents types of proof for task completion
type ProofType string

const (
	ProofTypePostURL      ProofType = "post_url"
	ProofTypeTxHash       ProofType = "tx_hash"
	ProofTypeCastHash     ProofType = "cast_hash"
	ProofTypeScreenshot   ProofType = "screenshot"
	ProofTypeAPIResponse  ProofType = "api_response"
	ProofTypeSignature    ProofType = "signature"
	ProofTypeManualVerify ProofType = "manual_verify"
)

// TaskProof represents proof of task completion
type TaskProof struct {
	Type       ProofType              `json:"type"`
	Value      string                 `json:"value"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Screenshot string                 `json:"screenshot,omitempty"`
	RawData    json.RawMessage        `json:"raw_data,omitempty"`
}

// TaskExecution represents a task execution record
type TaskExecution struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskID    uuid.UUID  `gorm:"type:uuid;not null" json:"task_id"`
	WalletID  *uuid.UUID `gorm:"type:uuid" json:"wallet_id,omitempty"`
	AccountID *uuid.UUID `gorm:"type:uuid" json:"account_id,omitempty"`

	// Status tracking
	Status      TaskStatus `gorm:"size:30;not null;default:'PENDING'" json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Idempotency
	IdempotencyKey string `gorm:"size:200;uniqueIndex;not null" json:"idempotency_key"`

	// Proof
	ProofType      string `gorm:"size:50" json:"proof_type,omitempty"`
	ProofValue     string `gorm:"size:500" json:"proof_value,omitempty"`
	ProofData      string `gorm:"type:jsonb" json:"proof_data,omitempty"`
	ScreenshotPath string `gorm:"size:500" json:"screenshot_path,omitempty"`

	// Results
	TransactionHash string `gorm:"size:100" json:"transaction_hash,omitempty"`
	PostID          string `gorm:"size:200" json:"post_id,omitempty"`
	PostURL         string `gorm:"size:500" json:"post_url,omitempty"`
	ResultData      string `gorm:"type:jsonb" json:"result_data,omitempty"`
	ErrorMessage    string `gorm:"type:text" json:"error_message,omitempty"`
	ErrorCode       string `gorm:"size:50" json:"error_code,omitempty"`

	// Browser session
	BrowserSessionID *uuid.UUID `gorm:"type:uuid" json:"browser_session_id,omitempty"`

	// Retry handling
	RetryCount  int        `gorm:"default:0" json:"retry_count"`
	MaxRetries  int        `gorm:"default:3" json:"max_retries"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`

	// Audit
	AuditLogID *uuid.UUID `gorm:"type:uuid" json:"audit_log_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExecutionRequest represents a request to execute a task
type ExecutionRequest struct {
	TaskID    uuid.UUID              `json:"task_id"`
	UserID    uuid.UUID              `json:"user_id"`
	WalletID  *uuid.UUID             `json:"wallet_id,omitempty"`
	AccountID *uuid.UUID             `json:"account_id,omitempty"`
	Force     bool                   `json:"force"` // Force even if already executed
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionResult represents the result of a task execution
type ExecutionResult struct {
	Execution *TaskExecution `json:"execution"`
	Proof     *TaskProof     `json:"proof,omitempty"`
	Error     error          `json:"-"`
}

// TaskExecutor is the interface for task executors
type TaskExecutor interface {
	Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error)
	CanHandle(taskType string) bool
}

// TaskManager manages task execution with idempotency and locking
type TaskManager struct {
	db          *gorm.DB
	lockManager *locks.LockManager
	taskQueue   *queue.Queue
	executors   map[string]TaskExecutor
}

// NewTaskManager creates a new task manager
func NewTaskManager(db *gorm.DB, lockManager *locks.LockManager, taskQueue *queue.Queue) *TaskManager {
	return &TaskManager{
		db:          db,
		lockManager: lockManager,
		taskQueue:   taskQueue,
		executors:   make(map[string]TaskExecutor),
	}
}

// RegisterExecutor registers a task executor for a task type
func (m *TaskManager) RegisterExecutor(taskType string, executor TaskExecutor) {
	m.executors[taskType] = executor
}

// GenerateIdempotencyKey generates a unique idempotency key for a task execution
func GenerateIdempotencyKey(taskID uuid.UUID, accountID, walletID *uuid.UUID, date time.Time) string {
	parts := taskID.String()
	if accountID != nil {
		parts += ":" + accountID.String()
	}
	if walletID != nil {
		parts += ":" + walletID.String()
	}
	parts += ":" + date.Format("2006-01-02")

	hash := sha256.Sum256([]byte(parts))
	return hex.EncodeToString(hash[:])
}

// Execute executes a task with idempotency checking and locking
func (m *TaskManager) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
	// Generate idempotency key
	idempotencyKey := GenerateIdempotencyKey(req.TaskID, req.AccountID, req.WalletID, time.Now())

	// Check for existing execution (idempotency)
	var existingExec TaskExecution
	err := m.db.Where("idempotency_key = ?", idempotencyKey).First(&existingExec).Error
	if err == nil && !req.Force {
		// Already executed
		if existingExec.Status == StatusDone {
			return &ExecutionResult{
				Execution: &existingExec,
			}, nil
		}
		// If failed/pending, allow re-execution
		if existingExec.Status != StatusFailed && existingExec.Status != StatusPending {
			return &ExecutionResult{
				Execution: &existingExec,
				Error:     errors.New("task already in progress"),
			}, nil
		}
	}

	// Determine lock resource
	var lockResource string
	var lockType locks.ResourceType
	if req.AccountID != nil {
		lockResource = req.AccountID.String()
		lockType = locks.ResourceAccount
	} else if req.WalletID != nil {
		lockResource = req.WalletID.String()
		lockType = locks.ResourceWallet
	} else {
		lockResource = req.TaskID.String()
		lockType = locks.ResourceTask
	}

	// Acquire lock
	lock, err := m.lockManager.AcquireWithRetry(ctx, lockType, lockResource, 5*time.Minute, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Release(ctx)

	// Create or update execution record
	now := time.Now()
	execution := &TaskExecution{
		ID:             uuid.New(),
		TaskID:         req.TaskID,
		WalletID:       req.WalletID,
		AccountID:      req.AccountID,
		Status:         StatusRunning,
		StartedAt:      &now,
		IdempotencyKey: idempotencyKey,
		MaxRetries:     3,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// If re-executing a failed task, update instead of create
	if existingExec.ID != uuid.Nil && (existingExec.Status == StatusFailed || existingExec.Status == StatusPending) {
		execution.ID = existingExec.ID
		execution.RetryCount = existingExec.RetryCount + 1
		execution.CreatedAt = existingExec.CreatedAt
		m.db.Save(execution)
	} else {
		if err := m.db.Create(execution).Error; err != nil {
			return nil, fmt.Errorf("failed to create execution record: %w", err)
		}
	}

	// Get task details to determine executor
	var task struct {
		Type           string
		RequiresManual bool
	}
	if err := m.db.Table("campaign_tasks").
		Select("type, requires_manual").
		Where("id = ?", req.TaskID).
		First(&task).Error; err != nil {
		execution.Status = StatusFailed
		execution.ErrorMessage = "Task not found"
		m.db.Save(execution)
		return &ExecutionResult{
			Execution: execution,
			Error:     err,
		}, nil
	}

	// Check if requires manual intervention
	if task.RequiresManual {
		execution.Status = StatusManualRequired
		m.db.Save(execution)
		return &ExecutionResult{
			Execution: execution,
		}, nil
	}

	// Find and execute
	executor, exists := m.executors[task.Type]
	if !exists {
		execution.Status = StatusFailed
		execution.ErrorMessage = fmt.Sprintf("No executor for task type: %s", task.Type)
		m.db.Save(execution)
		return &ExecutionResult{
			Execution: execution,
			Error:     fmt.Errorf("no executor for task type: %s", task.Type),
		}, nil
	}

	// Execute the task
	result, err := executor.Execute(ctx, req)
	if err != nil {
		execution.Status = StatusFailed
		execution.ErrorMessage = err.Error()

		// Schedule retry if not exceeded
		if execution.RetryCount < execution.MaxRetries {
			backoff := time.Duration(1<<uint(execution.RetryCount)) * time.Minute
			nextRetry := time.Now().Add(backoff)
			execution.NextRetryAt = &nextRetry

			// Queue retry
			m.taskQueue.Enqueue(ctx, "task_retry", map[string]interface{}{
				"execution_id": execution.ID,
				"task_id":      req.TaskID,
			}, queue.WithDelay(backoff))
		}

		m.db.Save(execution)
		return &ExecutionResult{
			Execution: execution,
			Error:     err,
		}, nil
	}

	// Success
	completedAt := time.Now()
	execution.Status = StatusDone
	execution.CompletedAt = &completedAt

	if result.Proof != nil {
		execution.ProofType = string(result.Proof.Type)
		execution.ProofValue = result.Proof.Value
		if proofData, err := json.Marshal(result.Proof); err == nil {
			execution.ProofData = string(proofData)
		}
		execution.ScreenshotPath = result.Proof.Screenshot
	}

	if result.Execution != nil {
		execution.TransactionHash = result.Execution.TransactionHash
		execution.PostID = result.Execution.PostID
		execution.PostURL = result.Execution.PostURL
		execution.ResultData = result.Execution.ResultData
	}

	m.db.Save(execution)

	return &ExecutionResult{
		Execution: execution,
		Proof:     result.Proof,
	}, nil
}

// ContinueManualTask continues a task that required manual intervention
func (m *TaskManager) ContinueManualTask(ctx context.Context, executionID uuid.UUID, proof *TaskProof) (*ExecutionResult, error) {
	var execution TaskExecution
	if err := m.db.First(&execution, executionID).Error; err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	if execution.Status != StatusManualRequired {
		return nil, errors.New("task is not waiting for manual input")
	}

	now := time.Now()
	execution.Status = StatusDone
	execution.CompletedAt = &now

	if proof != nil {
		execution.ProofType = string(proof.Type)
		execution.ProofValue = proof.Value
		if proofData, err := json.Marshal(proof); err == nil {
			execution.ProofData = string(proofData)
		}
		execution.ScreenshotPath = proof.Screenshot
	}

	if err := m.db.Save(&execution).Error; err != nil {
		return nil, err
	}

	return &ExecutionResult{
		Execution: &execution,
		Proof:     proof,
	}, nil
}

// GetExecution retrieves an execution by ID
func (m *TaskManager) GetExecution(ctx context.Context, executionID uuid.UUID) (*TaskExecution, error) {
	var execution TaskExecution
	if err := m.db.First(&execution, executionID).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// GetExecutionByIdempotencyKey retrieves an execution by idempotency key
func (m *TaskManager) GetExecutionByIdempotencyKey(ctx context.Context, key string) (*TaskExecution, error) {
	var execution TaskExecution
	if err := m.db.Where("idempotency_key = ?", key).First(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// ListExecutions lists executions for a task
func (m *TaskManager) ListExecutions(ctx context.Context, taskID uuid.UUID, status *TaskStatus, limit, offset int) ([]TaskExecution, int64, error) {
	var executions []TaskExecution
	var total int64

	query := m.db.Model(&TaskExecution{}).Where("task_id = ?", taskID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// CancelExecution cancels a pending or running execution
func (m *TaskManager) CancelExecution(ctx context.Context, executionID uuid.UUID) error {
	result := m.db.Model(&TaskExecution{}).
		Where("id = ? AND status IN ?", executionID, []TaskStatus{StatusPending, StatusRunning, StatusManualRequired}).
		Updates(map[string]interface{}{
			"status":        StatusSkipped,
			"error_message": "Cancelled by user",
			"updated_at":    time.Now(),
		})

	if result.RowsAffected == 0 {
		return errors.New("execution not found or cannot be cancelled")
	}
	return result.Error
}

// RetryExecution retries a failed execution
func (m *TaskManager) RetryExecution(ctx context.Context, executionID uuid.UUID) (*TaskExecution, error) {
	var execution TaskExecution
	if err := m.db.First(&execution, executionID).Error; err != nil {
		return nil, err
	}

	if execution.Status != StatusFailed {
		return nil, errors.New("only failed executions can be retried")
	}

	if execution.RetryCount >= execution.MaxRetries {
		return nil, errors.New("max retries exceeded")
	}

	// Reset for retry
	now := time.Now()
	execution.Status = StatusPending
	execution.RetryCount++
	execution.ErrorMessage = ""
	execution.ErrorCode = ""
	execution.UpdatedAt = now

	if err := m.db.Save(&execution).Error; err != nil {
		return nil, err
	}

	return &execution, nil
}
