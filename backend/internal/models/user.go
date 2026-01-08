package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email         string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash  string         `gorm:"size:255;not null" json:"-"`
	Name          string         `gorm:"size:100" json:"name"`
	
	// Settings
	Settings      string         `gorm:"type:jsonb" json:"settings"`
	
	// API Keys (encrypted)
	OpenAIKey     string         `gorm:"type:text" json:"-"`
	
	// Status
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	LastLoginAt   *time.Time     `json:"last_login_at,omitempty"`
	
	// Relations
	Wallets          []Wallet          `gorm:"foreignKey:UserID" json:"wallets,omitempty"`
	PlatformAccounts []PlatformAccount `gorm:"foreignKey:UserID" json:"platform_accounts,omitempty"`
	Campaigns        []Campaign        `gorm:"foreignKey:UserID" json:"campaigns,omitempty"`
	BrowserProfiles  []BrowserProfile  `gorm:"foreignKey:UserID" json:"browser_profiles,omitempty"`
	
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type Session struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Token        string    `gorm:"size:500;uniqueIndex;not null" json:"-"`
	RefreshToken string    `gorm:"size:500;uniqueIndex;not null" json:"-"`
	UserAgent    string    `gorm:"size:500" json:"user_agent"`
	IPAddress    string    `gorm:"size:50" json:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}
