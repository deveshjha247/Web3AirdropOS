package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CampaignType string

const (
	CampaignTypeAirdrop   CampaignType = "airdrop"
	CampaignTypeGalxe     CampaignType = "galxe"
	CampaignTypeZealy     CampaignType = "zealy"
	CampaignTypeLayer3    CampaignType = "layer3"
	CampaignTypeCustom    CampaignType = "custom"
	CampaignTypeFarcaster CampaignType = "farcaster"
)

type Campaign struct {
	ID           uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID        `gorm:"type:uuid;not null" json:"user_id"`
	Name         string           `gorm:"size:200;not null" json:"name"`
	Description  string           `gorm:"type:text" json:"description"`
	Type         CampaignType     `gorm:"size:50;not null" json:"type"`
	URL          string           `gorm:"size:500" json:"url"`
	ImageURL     string           `gorm:"size:500" json:"image_url"`
	
	// Timing
	StartDate    time.Time        `json:"start_date"`
	EndDate      time.Time        `json:"end_date"`
	Deadline     *time.Time       `json:"deadline,omitempty"`
	
	// Status
	Status       string           `gorm:"size:30;default:'active'" json:"status"` // active, paused, completed, expired
	Priority     int              `gorm:"default:0" json:"priority"`
	
	// Rewards
	EstimatedReward string        `gorm:"size:100" json:"estimated_reward"`
	RewardType      string        `gorm:"size:50" json:"reward_type"` // token, nft, points, unknown
	
	// Assignment
	WalletGroups []WalletGroup    `gorm:"many2many:campaign_wallet_groups;" json:"wallet_groups"`
	Tasks        []CampaignTask   `gorm:"foreignKey:CampaignID" json:"tasks"`
	
	// Progress
	TotalTasks      int           `json:"total_tasks"`
	CompletedTasks  int           `json:"completed_tasks"`
	ProgressPercent float64       `json:"progress_percent"`
	
	// Metadata
	Metadata     string           `gorm:"type:jsonb" json:"metadata"`
	
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	DeletedAt    gorm.DeletedAt   `gorm:"index" json:"-"`
}

type TaskType string

const (
	TaskTypeConnect      TaskType = "wallet_connect"
	TaskTypeTransaction  TaskType = "transaction"
	TaskTypeClaim        TaskType = "claim"
	TaskTypeFollow       TaskType = "follow"
	TaskTypeJoin         TaskType = "join"
	TaskTypePost         TaskType = "post"
	TaskTypeReply        TaskType = "reply"
	TaskTypeLike         TaskType = "like"
	TaskTypeRecast       TaskType = "recast"
	TaskTypeVerify       TaskType = "verify"
	TaskTypeQuiz         TaskType = "quiz"
	TaskTypeCustom       TaskType = "custom"
)

type CampaignTask struct {
	ID              uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CampaignID      uuid.UUID         `gorm:"type:uuid;not null" json:"campaign_id"`
	Name            string            `gorm:"size:200;not null" json:"name"`
	Description     string            `gorm:"type:text" json:"description"`
	Type            TaskType          `gorm:"size:50;not null" json:"type"`
	
	// Task details
	TargetURL       string            `gorm:"size:500" json:"target_url"`
	TargetPlatform  string            `gorm:"size:50" json:"target_platform"`
	TargetAccount   string            `gorm:"size:200" json:"target_account"` // username to follow, etc.
	RequiredAction  string            `gorm:"type:text" json:"required_action"` // detailed instructions
	
	// Automation
	IsAutomatable   bool              `gorm:"default:true" json:"is_automatable"`
	AutomationScript string           `gorm:"type:text" json:"automation_script,omitempty"`
	RequiresManual  bool              `gorm:"default:false" json:"requires_manual"` // needs human intervention
	
	// Order and dependencies
	Order           int               `gorm:"default:0" json:"order"`
	DependsOn       *uuid.UUID        `gorm:"type:uuid" json:"depends_on,omitempty"`
	
	// Points/Rewards
	Points          int               `json:"points"`
	
	// Execution tracking
	Executions      []TaskExecution   `gorm:"foreignKey:TaskID" json:"executions,omitempty"`
	
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type TaskExecution struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskID        uuid.UUID  `gorm:"type:uuid;not null" json:"task_id"`
	WalletID      *uuid.UUID `gorm:"type:uuid" json:"wallet_id,omitempty"`
	AccountID     *uuid.UUID `gorm:"type:uuid" json:"account_id,omitempty"`
	
	Status        string     `gorm:"size:30;not null" json:"status"` // pending, in_progress, waiting_manual, completed, failed, skipped
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	
	// Idempotency - prevents duplicate executions
	IdempotencyKey string    `gorm:"size:200;uniqueIndex" json:"idempotency_key"` // taskID+accountID+date or taskID+walletID+date
	
	// Proof of completion
	ProofType     string     `gorm:"size:50" json:"proof_type,omitempty"` // post_url, tx_hash, cast_hash, screenshot
	ProofValue    string     `gorm:"size:500" json:"proof_value,omitempty"`
	ProofData     string     `gorm:"type:jsonb" json:"proof_data,omitempty"` // Full proof object
	ScreenshotPath string    `gorm:"size:500" json:"screenshot_path,omitempty"`
	
	// Result
	TransactionHash string    `gorm:"size:100" json:"transaction_hash,omitempty"`
	PostID          string    `gorm:"size:200" json:"post_id,omitempty"`
	PostURL         string    `gorm:"size:500" json:"post_url,omitempty"`
	ResultData      string    `gorm:"type:jsonb" json:"result_data,omitempty"`
	ErrorMessage    string    `gorm:"type:text" json:"error_message,omitempty"`
	
	// Browser session
	BrowserSessionID *uuid.UUID `gorm:"type:uuid" json:"browser_session_id,omitempty"`
	
	// Retry info
	RetryCount    int        `gorm:"default:0" json:"retry_count"`
	MaxRetries    int        `gorm:"default:3" json:"max_retries"`
	
	// Audit link
	AuditLogID    *uuid.UUID `gorm:"type:uuid" json:"audit_log_id,omitempty"`
	
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
