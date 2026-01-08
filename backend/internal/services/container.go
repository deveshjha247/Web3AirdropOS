package services

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/config"
	"github.com/web3airdropos/backend/internal/services/platforms"
	"github.com/web3airdropos/backend/internal/websocket"
)

// Container holds all service instances
type Container struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	WSHub  *websocket.Hub

	// Core Services
	Auth      *AuthService
	Wallet    *WalletService
	Account   *AccountService
	Campaign  *CampaignService
	Task      *TaskService
	Browser   *BrowserService
	Content   *ContentService
	Job       *JobService
	Proxy     *ProxyService
	Dashboard *DashboardService

	// Production Services
	RateLimiter *RateLimiter
	Audit       *AuditService
}

func NewContainer(cfg *config.Config, db *gorm.DB, redis *redis.Client, wsHub *websocket.Hub) *Container {
	container := &Container{
		Config: cfg,
		DB:     db,
		Redis:  redis,
		WSHub:  wsHub,
	}

	// Initialize production services first (they have no dependencies)
	container.RateLimiter = NewRateLimiter(redis)
	container.Audit = NewAuditService(db)

	// Initialize all services
	container.Auth = NewAuthService(container)
	container.Wallet = NewWalletService(container)
	container.Account = NewAccountService(container)
	container.Campaign = NewCampaignService(container)
	container.Task = NewTaskService(container)
	container.Browser = NewBrowserService(container)
	container.Content = NewContentService(container)
	container.Job = NewJobService(container)
	container.Proxy = NewProxyService(container)
	container.Dashboard = NewDashboardService(container)

	// Register platform adapters with Task service
	container.registerPlatformAdapters(cfg)

	return container
}

// registerPlatformAdapters sets up platform adapters based on configuration
func (c *Container) registerPlatformAdapters(cfg *config.Config) {
	// Farcaster (Neynar)
	if cfg.NeynarAPIKey != "" {
		farcasterAdapter, err := platforms.NewFarcasterClient(&platforms.AccountCredentials{
			APIKey: cfg.NeynarAPIKey,
		})
		if err == nil {
			c.Task.RegisterAdapter("farcaster", farcasterAdapter)
		}
	}

	// Telegram
	if cfg.TelegramBotToken != "" {
		telegramAdapter, err := platforms.NewTelegramClient(&platforms.AccountCredentials{
			APIKey: cfg.TelegramBotToken,
		})
		if err == nil {
			c.Task.RegisterAdapter("telegram", telegramAdapter)
		}
	}

	// Twitter (skeleton - requires API access)
	if cfg.TwitterBearerToken != "" {
		twitterAdapter, err := platforms.NewTwitterClient(&platforms.AccountCredentials{
			APIKey:      cfg.TwitterAPIKey,
			APISecret:   cfg.TwitterSecret,
			AccessToken: cfg.TwitterBearerToken,
		})
		if err == nil {
			c.Task.RegisterAdapter("twitter", twitterAdapter)
			c.Task.RegisterAdapter("x", twitterAdapter)
		}
	}
}
