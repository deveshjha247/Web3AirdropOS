package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobType string

const (
	JobTypeScheduledPost   JobType = "scheduled_post"
	JobTypeCampaignTask    JobType = "campaign_task"
	JobTypeBalanceSync     JobType = "balance_sync"
	JobTypePlatformSync    JobType = "platform_sync"
	JobTypeEngagement      JobType = "engagement"
	JobTypeContentGenerate JobType = "content_generate"
	JobTypeBulkExecute     JobType = "bulk_execute"
)

type AutomationJob struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Type         JobType        `gorm:"size:50;not null" json:"type"`
	Name         string         `gorm:"size:200" json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	
	// Schedule
	CronExpression string       `gorm:"size:100" json:"cron_expression,omitempty"`
	NextRunAt      *time.Time   `json:"next_run_at,omitempty"`
	LastRunAt      *time.Time   `json:"last_run_at,omitempty"`
	
	// Status
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	Status       string         `gorm:"size:30" json:"status"` // idle, running, paused, failed
	
	// Configuration
	Config       string         `gorm:"type:jsonb" json:"config"` // job-specific configuration
	
	// Targeting
	WalletIDs    string         `gorm:"type:jsonb" json:"wallet_ids,omitempty"`   // array of wallet IDs
	AccountIDs   string         `gorm:"type:jsonb" json:"account_ids,omitempty"` // array of account IDs
	CampaignID   *uuid.UUID     `gorm:"type:uuid" json:"campaign_id,omitempty"`
	
	// Stats
	TotalRuns    int            `gorm:"default:0" json:"total_runs"`
	SuccessRuns  int            `gorm:"default:0" json:"success_runs"`
	FailedRuns   int            `gorm:"default:0" json:"failed_runs"`
	
	// Logs
	Logs         []JobLog       `gorm:"foreignKey:JobID" json:"logs,omitempty"`
	
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type JobLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	JobID     uuid.UUID `gorm:"type:uuid;not null" json:"job_id"`
	Level     string    `gorm:"size:20;not null" json:"level"` // info, warn, error, debug
	Message   string    `gorm:"type:text;not null" json:"message"`
	Details   string    `gorm:"type:jsonb" json:"details,omitempty"`
	
	// Context
	WalletID  *uuid.UUID `gorm:"type:uuid" json:"wallet_id,omitempty"`
	AccountID *uuid.UUID `gorm:"type:uuid" json:"account_id,omitempty"`
	TaskID    *uuid.UUID `gorm:"type:uuid" json:"task_id,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
}

type ContentDraft struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Platform    string     `gorm:"size:30" json:"platform"`
	Type        string     `gorm:"size:30" json:"type"` // post, reply, thread
	Content     string     `gorm:"type:text;not null" json:"content"`
	MediaURLs   string     `gorm:"type:jsonb" json:"media_urls,omitempty"`
	
	// AI generation info
	Prompt      string     `gorm:"type:text" json:"prompt,omitempty"`
	AIModel     string     `gorm:"size:50" json:"ai_model,omitempty"`
	Tone        string     `gorm:"size:30" json:"tone,omitempty"` // casual, professional, funny, etc.
	
	// Status
	Status      string     `gorm:"size:30;default:'draft'" json:"status"` // draft, approved, scheduled, posted
	
	// Engagement prediction
	PredictedEngagement string `gorm:"type:jsonb" json:"predicted_engagement,omitempty"`
	
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ScheduledPost struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	AccountID     uuid.UUID  `gorm:"type:uuid;not null" json:"account_id"`
	DraftID       *uuid.UUID `gorm:"type:uuid" json:"draft_id,omitempty"`
	
	Content       string     `gorm:"type:text;not null" json:"content"`
	MediaURLs     string     `gorm:"type:jsonb" json:"media_urls,omitempty"`
	Platform      string     `gorm:"size:50;not null" json:"platform"` // farcaster, x, telegram, discord
	
	// Reply context
	ReplyToID     string     `gorm:"size:200" json:"reply_to_id,omitempty"`
	ReplyToURL    string     `gorm:"size:500" json:"reply_to_url,omitempty"`
	
	// Schedule
	ScheduledFor  time.Time  `json:"scheduled_for"`
	ScheduledAt   time.Time  `json:"scheduled_at"` // Alias for compatibility
	TimeZone      string     `gorm:"size:50" json:"timezone"`
	
	// Status
	Status        string     `gorm:"size:30;default:'pending'" json:"status"` // pending, posted, failed, cancelled
	PostedAt      *time.Time `json:"posted_at,omitempty"`
	PostID        string     `gorm:"size:200" json:"post_id,omitempty"` // ID of the actual post
	PostURL       string     `gorm:"size:500" json:"post_url,omitempty"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message,omitempty"`
	
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
