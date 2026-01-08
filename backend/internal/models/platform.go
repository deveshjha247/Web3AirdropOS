package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlatformType string

const (
	PlatformFarcaster PlatformType = "farcaster"
	PlatformTwitter   PlatformType = "twitter"
	PlatformTelegram  PlatformType = "telegram"
	PlatformDiscord   PlatformType = "discord"
)

type PlatformAccount struct {
	ID               uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID         `gorm:"type:uuid;not null" json:"user_id"`
	Platform         PlatformType      `gorm:"size:30;not null" json:"platform"`
	Username         string            `gorm:"size:100" json:"username"`
	DisplayName      string            `gorm:"size:200" json:"display_name"`
	PlatformUserID   string            `gorm:"size:100" json:"platform_user_id"`
	ProfileURL       string            `gorm:"size:500" json:"profile_url"`
	AvatarURL        string            `gorm:"size:500" json:"avatar_url"`
	WalletID         *uuid.UUID        `gorm:"type:uuid" json:"wallet_id,omitempty"`
	BrowserProfileID *uuid.UUID        `gorm:"type:uuid" json:"browser_profile_id,omitempty"`
	
	// Authentication
	AccessToken      string            `gorm:"type:text" json:"-"`
	RefreshToken     string            `gorm:"type:text" json:"-"`
	TokenExpiry      time.Time         `json:"token_expiry"`
	Cookies          string            `gorm:"type:text" json:"-"` // Encrypted cookies for browser session
	
	// Status
	IsActive         bool              `gorm:"default:true" json:"is_active"`
	LastLoginAt      time.Time         `json:"last_login_at"`
	LastActivityAt   time.Time         `json:"last_activity_at"`
	
	// Stats
	FollowerCount    int               `json:"follower_count"`
	FollowingCount   int               `json:"following_count"`
	PostCount        int               `json:"post_count"`
	
	// Proxy settings
	ProxyID          *uuid.UUID        `gorm:"type:uuid" json:"proxy_id,omitempty"`
	
	// Relations
	Activities       []AccountActivity `gorm:"foreignKey:AccountID" json:"activities,omitempty"`
	ScheduledPosts   []ScheduledPost   `gorm:"foreignKey:AccountID" json:"scheduled_posts,omitempty"`
	
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	DeletedAt        gorm.DeletedAt    `gorm:"index" json:"-"`
}

type AccountActivity struct {
	ID          uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AccountID   uuid.UUID    `gorm:"type:uuid;not null" json:"account_id"`
	Type        string       `gorm:"size:50;not null" json:"type"` // post, reply, like, follow, recast, etc.
	TargetID    string       `gorm:"size:200" json:"target_id"`    // ID of post/user targeted
	TargetURL   string       `gorm:"size:500" json:"target_url"`
	Content     string       `gorm:"type:text" json:"content"`
	Metadata    string       `gorm:"type:jsonb" json:"metadata"`
	Status      string       `gorm:"size:30" json:"status"` // success, failed, pending
	ErrorMsg    string       `gorm:"type:text" json:"error_msg,omitempty"`
	CampaignID  *uuid.UUID   `gorm:"type:uuid" json:"campaign_id,omitempty"`
	AutomatedBy string       `gorm:"size:50" json:"automated_by"` // manual, scheduled, ai, campaign
	CreatedAt   time.Time    `json:"created_at"`
}

type Proxy struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Name      string         `gorm:"size:100" json:"name"`
	Type      string         `gorm:"size:20" json:"type"` // http, socks5, residential
	Host      string         `gorm:"size:200;not null" json:"host"`
	Port      int            `gorm:"not null" json:"port"`
	Username  string         `gorm:"size:100" json:"username"`
	Password  string         `gorm:"size:200" json:"-"`
	Country   string         `gorm:"size:10" json:"country"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	LastCheck time.Time      `json:"last_check"`
	Latency   int            `json:"latency"` // in milliseconds
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
