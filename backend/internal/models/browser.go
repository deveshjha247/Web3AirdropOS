package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrowserProfile struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Name            string         `gorm:"size:100;not null" json:"name"`
	
	// Browser fingerprint settings
	UserAgent       string         `gorm:"size:500" json:"user_agent"`
	ScreenWidth     int            `gorm:"default:1920" json:"screen_width"`
	ScreenHeight    int            `gorm:"default:1080" json:"screen_height"`
	Language        string         `gorm:"size:20;default:'en-US'" json:"language"`
	Timezone        string         `gorm:"size:50" json:"timezone"`
	Platform        string         `gorm:"size:50" json:"platform"` // Windows, MacOS, Linux
	
	// WebGL/Canvas fingerprint
	WebGLVendor     string         `gorm:"size:200" json:"webgl_vendor"`
	WebGLRenderer   string         `gorm:"size:200" json:"webgl_renderer"`
	
	// Proxy settings
	ProxyID         *uuid.UUID     `gorm:"type:uuid" json:"proxy_id,omitempty"`
	
	// Cookie storage path
	CookiePath      string         `gorm:"size:500" json:"cookie_path"`
	LocalStoragePath string        `gorm:"size:500" json:"local_storage_path"`
	
	// Linked accounts
	PlatformAccounts []PlatformAccount `gorm:"foreignKey:BrowserProfileID" json:"platform_accounts,omitempty"`
	
	// Status
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	LastUsedAt      *time.Time     `json:"last_used_at,omitempty"`
	
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type BrowserSession struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	ProfileID       uuid.UUID  `gorm:"type:uuid;not null" json:"profile_id"`
	
	// Container info
	ContainerID     string     `gorm:"size:100" json:"container_id"`
	VNCURL          string     `gorm:"size:200" json:"vnc_url"`
	DebuggerURL     string     `gorm:"size:200" json:"debugger_url"`
	WebSocketURL    string     `gorm:"size:200" json:"websocket_url"`
	
	// Current state
	CurrentURL      string     `gorm:"size:1000" json:"current_url"`
	CurrentTitle    string     `gorm:"size:500" json:"current_title"`
	
	// Task context
	TaskExecutionID *uuid.UUID `gorm:"type:uuid" json:"task_execution_id,omitempty"`
	
	// Status
	Status          string     `gorm:"size:30" json:"status"` // starting, ready, busy, waiting_manual, stopped
	ManualRequired  bool       `gorm:"default:false" json:"manual_required"`
	ManualMessage   string     `gorm:"type:text" json:"manual_message,omitempty"`
	
	// Timing
	StartedAt       time.Time  `json:"started_at"`
	LastActivityAt  time.Time  `json:"last_activity_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type BrowserAction struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID uuid.UUID `gorm:"type:uuid;not null" json:"session_id"`
	
	// Action details
	Type      string    `gorm:"size:50;not null" json:"type"` // navigate, click, type, scroll, screenshot, etc.
	Target    string    `gorm:"size:500" json:"target"`       // CSS selector or URL
	Value     string    `gorm:"type:text" json:"value"`       // text to type, etc.
	
	// Result
	Status    string    `gorm:"size:30" json:"status"` // pending, success, failed
	Result    string    `gorm:"type:jsonb" json:"result,omitempty"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	
	// Screenshot
	ScreenshotPath string `gorm:"size:500" json:"screenshot_path,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
}
