package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Action represents an auditable action
type Action string

const (
	// Social actions
	ActionFollow     Action = "follow"
	ActionUnfollow   Action = "unfollow"
	ActionLike       Action = "like"
	ActionUnlike     Action = "unlike"
	ActionRepost     Action = "repost"
	ActionPost       Action = "post"
	ActionReply      Action = "reply"
	ActionQuote      Action = "quote"
	ActionDelete     Action = "delete"

	// Wallet actions
	ActionTransaction   Action = "transaction"
	ActionTokenApproval Action = "token_approval"
	ActionSwap          Action = "swap"
	ActionBridge        Action = "bridge"
	ActionMint          Action = "mint"
	ActionClaim         Action = "claim"

	// Content actions
	ActionGenerate Action = "generate"
	ActionApprove  Action = "approve"
	ActionReject   Action = "reject"
	ActionSchedule Action = "schedule"
	ActionPublish  Action = "publish"

	// Account actions
	ActionLogin        Action = "login"
	ActionLogout       Action = "logout"
	ActionRegister     Action = "register"
	ActionPasswordChange Action = "password_change"
	ActionAccountLink  Action = "account_link"
	ActionWalletCreate Action = "wallet_create"
	ActionWalletImport Action = "wallet_import"
	ActionSecretAccess Action = "secret_access"
	ActionSecretCreate Action = "secret_create"
	ActionSecretDelete Action = "secret_delete"

	// System actions
	ActionTaskStart    Action = "task_start"
	ActionTaskComplete Action = "task_complete"
	ActionTaskFail     Action = "task_fail"
	ActionTaskRetry    Action = "task_retry"
	ActionJobRun       Action = "job_run"
	ActionJobComplete  Action = "job_complete"
	ActionJobFail      Action = "job_fail"
	ActionBrowserAction Action = "browser_action"
	ActionAPIRequest   Action = "api_request"
)

// Result represents the outcome of an action
type Result string

const (
	ResultSuccess Result = "success"
	ResultFailed  Result = "failed"
	ResultPending Result = "pending"
	ResultSkipped Result = "skipped"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	
	// Who
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountID      *uuid.UUID `gorm:"type:uuid;index" json:"account_id,omitempty"`
	WalletID       *uuid.UUID `gorm:"type:uuid;index" json:"wallet_id,omitempty"`
	ProfileID      *uuid.UUID `gorm:"type:uuid" json:"profile_id,omitempty"`
	
	// What
	Action         Action     `gorm:"size:50;not null;index" json:"action"`
	Platform       string     `gorm:"size:30;index" json:"platform,omitempty"`
	TargetType     string     `gorm:"size:50" json:"target_type,omitempty"`
	TargetID       string     `gorm:"size:200" json:"target_id,omitempty"`
	
	// Context
	TaskID         *uuid.UUID `gorm:"type:uuid;index" json:"task_id,omitempty"`
	ExecutionID    *uuid.UUID `gorm:"type:uuid;index" json:"execution_id,omitempty"`
	JobID          *uuid.UUID `gorm:"type:uuid;index" json:"job_id,omitempty"`
	CampaignID     *uuid.UUID `gorm:"type:uuid;index" json:"campaign_id,omitempty"`
	SessionID      *uuid.UUID `gorm:"type:uuid" json:"session_id,omitempty"`
	
	// Result
	Result         Result     `gorm:"size:20;not null;index" json:"result"`
	ErrorCode      string     `gorm:"size:50" json:"error_code,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	
	// Proof
	ProofType      string     `gorm:"size:50" json:"proof_type,omitempty"`
	ProofValue     string     `gorm:"size:500" json:"proof_value,omitempty"`
	ProofData      string     `gorm:"type:jsonb" json:"proof_data,omitempty"`
	
	// Request/Response for debugging
	RequestData    string     `gorm:"type:jsonb" json:"request_data,omitempty"`
	ResponseData   string     `gorm:"type:jsonb" json:"response_data,omitempty"`
	
	// Metadata
	IPAddress      string     `gorm:"size:50" json:"ip_address,omitempty"`
	UserAgent      string     `gorm:"size:300" json:"user_agent,omitempty"`
	DurationMs     int64      `json:"duration_ms,omitempty"`
	
	// Idempotency
	IdempotencyKey string     `gorm:"size:200;uniqueIndex" json:"idempotency_key,omitempty"`
	
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
}

// LogEntry represents a log entry to be created
type LogEntry struct {
	UserID         uuid.UUID
	AccountID      *uuid.UUID
	WalletID       *uuid.UUID
	ProfileID      *uuid.UUID
	
	Action         Action
	Platform       string
	TargetType     string
	TargetID       string
	
	TaskID         *uuid.UUID
	ExecutionID    *uuid.UUID
	JobID          *uuid.UUID
	CampaignID     *uuid.UUID
	SessionID      *uuid.UUID
	
	Result         Result
	ErrorCode      string
	ErrorMessage   string
	
	ProofType      string
	ProofValue     string
	ProofData      interface{}
	
	RequestData    interface{}
	ResponseData   interface{}
	
	Duration       time.Duration
	IPAddress      string
	UserAgent      string
	
	IdempotencyKey string
}

// Logger handles audit logging
type Logger struct {
	db        *gorm.DB
	batchSize int
	batch     chan *AuditLog
	stop      chan struct{}
}

// NewLogger creates a new audit logger
func NewLogger(db *gorm.DB) *Logger {
	logger := &Logger{
		db:        db,
		batchSize: 100,
		batch:     make(chan *AuditLog, 1000),
		stop:      make(chan struct{}),
	}
	
	// Start background batch processor
	go logger.processBatch()
	
	return logger
}

// Log creates an audit log entry
func (l *Logger) Log(ctx context.Context, entry *LogEntry) (*AuditLog, error) {
	log := &AuditLog{
		ID:             uuid.New(),
		UserID:         entry.UserID,
		AccountID:      entry.AccountID,
		WalletID:       entry.WalletID,
		ProfileID:      entry.ProfileID,
		Action:         entry.Action,
		Platform:       entry.Platform,
		TargetType:     entry.TargetType,
		TargetID:       entry.TargetID,
		TaskID:         entry.TaskID,
		ExecutionID:    entry.ExecutionID,
		JobID:          entry.JobID,
		CampaignID:     entry.CampaignID,
		SessionID:      entry.SessionID,
		Result:         entry.Result,
		ErrorCode:      entry.ErrorCode,
		ErrorMessage:   entry.ErrorMessage,
		ProofType:      entry.ProofType,
		ProofValue:     entry.ProofValue,
		IPAddress:      entry.IPAddress,
		UserAgent:      entry.UserAgent,
		DurationMs:     entry.Duration.Milliseconds(),
		IdempotencyKey: entry.IdempotencyKey,
		CreatedAt:      time.Now(),
	}

	// Serialize JSON fields
	if entry.ProofData != nil {
		if data, err := json.Marshal(entry.ProofData); err == nil {
			log.ProofData = string(data)
		}
	}
	if entry.RequestData != nil {
		if data, err := json.Marshal(entry.RequestData); err == nil {
			log.RequestData = string(data)
		}
	}
	if entry.ResponseData != nil {
		if data, err := json.Marshal(entry.ResponseData); err == nil {
			log.ResponseData = string(data)
		}
	}

	// Send to batch processor (non-blocking)
	select {
	case l.batch <- log:
	default:
		// Batch channel full - write directly
		if err := l.db.Create(log).Error; err != nil {
			return nil, err
		}
	}

	return log, nil
}

// LogSync creates an audit log entry synchronously (guaranteed write)
func (l *Logger) LogSync(ctx context.Context, entry *LogEntry) (*AuditLog, error) {
	log := &AuditLog{
		ID:             uuid.New(),
		UserID:         entry.UserID,
		AccountID:      entry.AccountID,
		WalletID:       entry.WalletID,
		ProfileID:      entry.ProfileID,
		Action:         entry.Action,
		Platform:       entry.Platform,
		TargetType:     entry.TargetType,
		TargetID:       entry.TargetID,
		TaskID:         entry.TaskID,
		ExecutionID:    entry.ExecutionID,
		JobID:          entry.JobID,
		CampaignID:     entry.CampaignID,
		SessionID:      entry.SessionID,
		Result:         entry.Result,
		ErrorCode:      entry.ErrorCode,
		ErrorMessage:   entry.ErrorMessage,
		ProofType:      entry.ProofType,
		ProofValue:     entry.ProofValue,
		IPAddress:      entry.IPAddress,
		UserAgent:      entry.UserAgent,
		DurationMs:     entry.Duration.Milliseconds(),
		IdempotencyKey: entry.IdempotencyKey,
		CreatedAt:      time.Now(),
	}

	// Serialize JSON fields
	if entry.ProofData != nil {
		if data, err := json.Marshal(entry.ProofData); err == nil {
			log.ProofData = string(data)
		}
	}
	if entry.RequestData != nil {
		if data, err := json.Marshal(entry.RequestData); err == nil {
			log.RequestData = string(data)
		}
	}
	if entry.ResponseData != nil {
		if data, err := json.Marshal(entry.ResponseData); err == nil {
			log.ResponseData = string(data)
		}
	}

	if err := l.db.Create(log).Error; err != nil {
		return nil, err
	}

	return log, nil
}

// LogSuccess is a convenience method for successful actions
func (l *Logger) LogSuccess(ctx context.Context, userID uuid.UUID, action Action, platform string) (*AuditLog, error) {
	return l.Log(ctx, &LogEntry{
		UserID:   userID,
		Action:   action,
		Platform: platform,
		Result:   ResultSuccess,
	})
}

// LogFailure is a convenience method for failed actions
func (l *Logger) LogFailure(ctx context.Context, userID uuid.UUID, action Action, platform string, err error) (*AuditLog, error) {
	return l.Log(ctx, &LogEntry{
		UserID:       userID,
		Action:       action,
		Platform:     platform,
		Result:       ResultFailed,
		ErrorMessage: err.Error(),
	})
}

// processBatch processes batched log entries
func (l *Logger) processBatch() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var batch []*AuditLog

	flush := func() {
		if len(batch) == 0 {
			return
		}
		
		// Batch insert
		if err := l.db.CreateInBatches(batch, l.batchSize).Error; err != nil {
			// On error, try one by one
			for _, log := range batch {
				l.db.Create(log)
			}
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-l.stop:
			flush()
			return
		case log := <-l.batch:
			batch = append(batch, log)
			if len(batch) >= l.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Stop stops the audit logger
func (l *Logger) Stop() {
	close(l.stop)
}

// QueryParams represents query parameters for audit logs
type QueryParams struct {
	UserID     *uuid.UUID
	AccountID  *uuid.UUID
	WalletID   *uuid.UUID
	Action     *Action
	Platform   string
	Result     *Result
	CampaignID *uuid.UUID
	TaskID     *uuid.UUID
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}

// Query queries audit logs with filters
func (l *Logger) Query(ctx context.Context, params *QueryParams) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	query := l.db.Model(&AuditLog{})

	if params.UserID != nil {
		query = query.Where("user_id = ?", *params.UserID)
	}
	if params.AccountID != nil {
		query = query.Where("account_id = ?", *params.AccountID)
	}
	if params.WalletID != nil {
		query = query.Where("wallet_id = ?", *params.WalletID)
	}
	if params.Action != nil {
		query = query.Where("action = ?", *params.Action)
	}
	if params.Platform != "" {
		query = query.Where("platform = ?", params.Platform)
	}
	if params.Result != nil {
		query = query.Where("result = ?", *params.Result)
	}
	if params.CampaignID != nil {
		query = query.Where("campaign_id = ?", *params.CampaignID)
	}
	if params.TaskID != nil {
		query = query.Where("task_id = ?", *params.TaskID)
	}
	if params.StartTime != nil {
		query = query.Where("created_at >= ?", *params.StartTime)
	}
	if params.EndTime != nil {
		query = query.Where("created_at <= ?", *params.EndTime)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	if err := query.Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByID retrieves an audit log by ID
func (l *Logger) GetByID(ctx context.Context, id uuid.UUID) (*AuditLog, error) {
	var log AuditLog
	if err := l.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetUserActivity returns activity summary for a user
func (l *Logger) GetUserActivity(ctx context.Context, userID uuid.UUID, days int) (map[Action]int64, error) {
	since := time.Now().AddDate(0, 0, -days)
	
	var results []struct {
		Action Action
		Count  int64
	}
	
	if err := l.db.Model(&AuditLog{}).
		Select("action, count(*) as count").
		Where("user_id = ? AND created_at >= ?", userID, since).
		Group("action").
		Find(&results).Error; err != nil {
		return nil, err
	}

	activity := make(map[Action]int64)
	for _, r := range results {
		activity[r.Action] = r.Count
	}
	
	return activity, nil
}

// Cleanup removes old audit logs based on retention policy
func (l *Logger) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := l.db.Where("created_at < ?", cutoff).Delete(&AuditLog{})
	return result.RowsAffected, result.Error
}
