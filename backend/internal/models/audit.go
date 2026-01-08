package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditLogAction represents the type of action being logged
type AuditLogAction string

const (
	// Social actions
	ActionFollow     AuditLogAction = "follow"
	ActionUnfollow   AuditLogAction = "unfollow"
	ActionLike       AuditLogAction = "like"
	ActionUnlike     AuditLogAction = "unlike"
	ActionRepost     AuditLogAction = "repost"
	ActionPost       AuditLogAction = "post"
	ActionReply      AuditLogAction = "reply"
	ActionQuote      AuditLogAction = "quote"
	ActionDelete     AuditLogAction = "delete"
	
	// Wallet actions
	ActionTransaction   AuditLogAction = "transaction"
	ActionTokenApproval AuditLogAction = "token_approval"
	ActionSwap          AuditLogAction = "swap"
	ActionBridge        AuditLogAction = "bridge"
	ActionMint          AuditLogAction = "mint"
	ActionClaim         AuditLogAction = "claim"
	
	// Content actions
	ActionGenerate AuditLogAction = "generate"
	ActionApprove  AuditLogAction = "approve"
	ActionReject   AuditLogAction = "reject"
	ActionSchedule AuditLogAction = "schedule"
	ActionPublish  AuditLogAction = "publish"
	
	// Account actions
	ActionLogin        AuditLogAction = "login"
	ActionAccountLink  AuditLogAction = "account_link"
	ActionWalletCreate AuditLogAction = "wallet_create"
	ActionWalletImport AuditLogAction = "wallet_import"
	
	// System actions
	ActionTaskStart    AuditLogAction = "task_start"
	ActionTaskComplete AuditLogAction = "task_complete"
	ActionTaskFail     AuditLogAction = "task_fail"
	ActionJobRun       AuditLogAction = "job_run"
	ActionBrowserAction AuditLogAction = "browser_action"
)

// AuditLogResult represents the outcome of an action
type AuditLogResult string

const (
	ResultSuccess AuditLogResult = "success"
	ResultFailed  AuditLogResult = "failed"
	ResultPending AuditLogResult = "pending"
	ResultSkipped AuditLogResult = "skipped"
)

// AuditLog stores comprehensive action logs for debugging and compliance
type AuditLog struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	
	// Who
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountID   *uuid.UUID     `gorm:"type:uuid;index" json:"account_id,omitempty"`   // Platform account
	WalletID    *uuid.UUID     `gorm:"type:uuid;index" json:"wallet_id,omitempty"`    // Wallet used
	ProfileID   *uuid.UUID     `gorm:"type:uuid" json:"profile_id,omitempty"`         // Browser profile
	
	// What
	Action      AuditLogAction `gorm:"size:50;not null;index" json:"action"`
	Platform    string         `gorm:"size:30;index" json:"platform,omitempty"` // farcaster, telegram, etc.
	TargetType  string         `gorm:"size:50" json:"target_type,omitempty"`    // user, post, transaction, etc.
	TargetID    string         `gorm:"size:200" json:"target_id,omitempty"`     // ID of the target
	
	// Context
	TaskID      *uuid.UUID     `gorm:"type:uuid;index" json:"task_id,omitempty"`
	ExecutionID *uuid.UUID     `gorm:"type:uuid;index" json:"execution_id,omitempty"`
	JobID       *uuid.UUID     `gorm:"type:uuid;index" json:"job_id,omitempty"`
	CampaignID  *uuid.UUID     `gorm:"type:uuid;index" json:"campaign_id,omitempty"`
	SessionID   *uuid.UUID     `gorm:"type:uuid" json:"session_id,omitempty"` // Browser session
	
	// Result
	Result      AuditLogResult `gorm:"size:20;not null;index" json:"result"`
	ErrorCode   string         `gorm:"size:50" json:"error_code,omitempty"`
	ErrorMessage string        `gorm:"type:text" json:"error_message,omitempty"`
	
	// Proof
	ProofType   string         `gorm:"size:50" json:"proof_type,omitempty"` // post_url, tx_hash, screenshot, etc.
	ProofValue  string         `gorm:"size:500" json:"proof_value,omitempty"`
	ProofData   string         `gorm:"type:jsonb" json:"proof_data,omitempty"` // Full proof JSON
	
	// Request/Response for debugging
	RequestData  string        `gorm:"type:jsonb" json:"request_data,omitempty"`
	ResponseData string        `gorm:"type:jsonb" json:"response_data,omitempty"`
	
	// Metadata
	IPAddress   string         `gorm:"size:50" json:"ip_address,omitempty"`
	UserAgent   string         `gorm:"size:300" json:"user_agent,omitempty"`
	Duration    int64          `json:"duration_ms,omitempty"` // Execution time in milliseconds
	
	// Idempotency
	IdempotencyKey string      `gorm:"size:200;uniqueIndex" json:"idempotency_key,omitempty"`
	
	CreatedAt   time.Time      `gorm:"index" json:"created_at"`
}

// AuditLogQuery represents query parameters for filtering audit logs
type AuditLogQuery struct {
	UserID      *uuid.UUID
	AccountID   *uuid.UUID
	WalletID    *uuid.UUID
	Action      *AuditLogAction
	Platform    string
	Result      *AuditLogResult
	CampaignID  *uuid.UUID
	TaskID      *uuid.UUID
	StartTime   *time.Time
	EndTime     *time.Time
	Limit       int
	Offset      int
}
