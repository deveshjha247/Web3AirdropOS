package platforms

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// PlatformType represents supported social platforms
type PlatformType string

const (
	PlatformFarcaster PlatformType = "farcaster"
	PlatformTwitter   PlatformType = "twitter"
	PlatformTelegram  PlatformType = "telegram"
	PlatformDiscord   PlatformType = "discord"
)

// Common errors
var (
	ErrNotImplemented     = errors.New("feature not implemented for this platform")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrRateLimited        = errors.New("rate limited by platform")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrPostNotFound       = errors.New("post not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrAlreadyFollowing   = errors.New("already following this user")
	ErrAlreadyLiked       = errors.New("already liked this post")
)

// ActionProof contains proof of a completed action
type ActionProof struct {
	PostID       string            `json:"post_id,omitempty"`
	PostURL      string            `json:"post_url,omitempty"`
	CastHash     string            `json:"cast_hash,omitempty"`
	TxHash       string            `json:"tx_hash,omitempty"`
	ScreenshotPath string          `json:"screenshot_path,omitempty"`
	Timestamp    int64             `json:"timestamp"`
	RawResponse  string            `json:"raw_response,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PostContent represents content to be posted
type PostContent struct {
	Text       string   `json:"text"`
	MediaURLs  []string `json:"media_urls,omitempty"`
	ReplyToID  string   `json:"reply_to_id,omitempty"`
	QuoteID    string   `json:"quote_id,omitempty"`
	ChannelID  string   `json:"channel_id,omitempty"` // For Farcaster channels
	EmbedURLs  []string `json:"embed_urls,omitempty"`
}

// UserProfile represents a user profile on any platform
type UserProfile struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	Verified    bool   `json:"verified"`
}

// PlatformAdapter is the interface all platform clients must implement
type PlatformAdapter interface {
	// Platform info
	GetPlatformType() PlatformType
	
	// Authentication
	Authenticate(ctx context.Context, credentials map[string]string) error
	IsAuthenticated() bool
	RefreshAuth(ctx context.Context) error
	
	// Profile operations
	GetProfile(ctx context.Context) (*UserProfile, error)
	GetUserByUsername(ctx context.Context, username string) (*UserProfile, error)
	
	// Social actions
	Follow(ctx context.Context, targetUserID string) (*ActionProof, error)
	Unfollow(ctx context.Context, targetUserID string) (*ActionProof, error)
	Like(ctx context.Context, postID string) (*ActionProof, error)
	Unlike(ctx context.Context, postID string) (*ActionProof, error)
	Repost(ctx context.Context, postID string) (*ActionProof, error) // Recast/Retweet
	
	// Content operations
	Post(ctx context.Context, content *PostContent) (*ActionProof, error)
	Reply(ctx context.Context, postID string, content *PostContent) (*ActionProof, error)
	Quote(ctx context.Context, postID string, content *PostContent) (*ActionProof, error)
	DeletePost(ctx context.Context, postID string) error
	
	// Verification
	VerifyAction(ctx context.Context, actionType string, proof *ActionProof) (bool, error)
	
	// Rate limit info
	GetRateLimitStatus(ctx context.Context) (*RateLimitStatus, error)
}

// RateLimitStatus contains rate limit information
type RateLimitStatus struct {
	Remaining   int   `json:"remaining"`
	Limit       int   `json:"limit"`
	ResetAt     int64 `json:"reset_at"`
	RetryAfter  int   `json:"retry_after,omitempty"`
}

// AccountCredentials holds platform-specific credentials
type AccountCredentials struct {
	AccountID    uuid.UUID         `json:"account_id"`
	Platform     PlatformType      `json:"platform"`
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	APIKey       string            `json:"api_key,omitempty"`
	APISecret    string            `json:"api_secret,omitempty"`
	PrivateKey   string            `json:"private_key,omitempty"` // For Farcaster signer
	FID          uint64            `json:"fid,omitempty"`         // Farcaster ID
	ExpiresAt    int64             `json:"expires_at,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// AdapterFactory creates platform adapters
type AdapterFactory struct {
	httpTimeout int
}

func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{
		httpTimeout: 30,
	}
}

func (f *AdapterFactory) CreateAdapter(creds *AccountCredentials) (PlatformAdapter, error) {
	switch creds.Platform {
	case PlatformFarcaster:
		return NewFarcasterClient(creds)
	case PlatformTelegram:
		return NewTelegramClient(creds)
	case PlatformTwitter:
		return NewTwitterClient(creds)
	case PlatformDiscord:
		return nil, errors.New("discord adapter is notification-only, no user automation")
	default:
		return nil, errors.New("unsupported platform: " + string(creds.Platform))
	}
}
