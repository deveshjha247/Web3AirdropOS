package services

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/config"
	"github.com/web3airdropos/backend/internal/websocket"
)

// Container holds all service instances
type Container struct {
	Config     *config.Config
	DB         *gorm.DB
	Redis      *redis.Client
	WSHub      *websocket.Hub
	
	// Services
	Auth       *AuthService
	Wallet     *WalletService
	Account    *AccountService
	Campaign   *CampaignService
	Task       *TaskService
	Browser    *BrowserService
	Content    *ContentService
	Job        *JobService
	Proxy      *ProxyService
	Dashboard  *DashboardService
}

func NewContainer(cfg *config.Config, db *gorm.DB, redis *redis.Client, wsHub *websocket.Hub) *Container {
	container := &Container{
		Config: cfg,
		DB:     db,
		Redis:  redis,
		WSHub:  wsHub,
	}

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

	return container
}
