package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/cryptoautomation/backend/internal/api/handlers"
	"github.com/cryptoautomation/backend/internal/api/middleware"
	"github.com/cryptoautomation/backend/internal/config"
	"github.com/cryptoautomation/backend/internal/services"
	"github.com/cryptoautomation/backend/internal/websocket"
)

type Server struct {
	router      *gin.Engine
	config      *config.Config
	db          *gorm.DB
	redis       *redis.Client
	wsHub       *websocket.Hub
	services    *services.Container
}

func NewServer(cfg *config.Config, db *gorm.DB, redis *redis.Client, wsHub *websocket.Hub) *Server {
	// Initialize services container
	svc := services.NewContainer(cfg, db, redis, wsHub)

	server := &Server{
		router:   gin.Default(),
		config:   cfg,
		db:       db,
		redis:    redis,
		wsHub:    wsHub,
		services: svc,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// CORS middleware
	s.router.Use(middleware.CORS())
	
	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "crypto-automation-backend",
		})
	})

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			authHandler := handlers.NewAuthHandler(s.services)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.Auth(s.config.JWTSecret))
		{
			// Wallet routes
			wallets := protected.Group("/wallets")
			{
				walletHandler := handlers.NewWalletHandler(s.services)
				wallets.GET("", walletHandler.List)
				wallets.POST("", walletHandler.Create)
				wallets.GET("/:id", walletHandler.Get)
				wallets.PUT("/:id", walletHandler.Update)
				wallets.DELETE("/:id", walletHandler.Delete)
				wallets.GET("/:id/balance", walletHandler.GetBalance)
				wallets.GET("/:id/transactions", walletHandler.GetTransactions)
				wallets.POST("/:id/prepare-tx", walletHandler.PrepareTransaction)
				wallets.POST("/import", walletHandler.Import)
				wallets.POST("/bulk", walletHandler.BulkCreate)
			}

			// Wallet groups
			groups := protected.Group("/wallet-groups")
			{
				groupHandler := handlers.NewWalletGroupHandler(s.services)
				groups.GET("", groupHandler.List)
				groups.POST("", groupHandler.Create)
				groups.PUT("/:id", groupHandler.Update)
				groups.DELETE("/:id", groupHandler.Delete)
				groups.POST("/:id/wallets", groupHandler.AddWallets)
				groups.DELETE("/:id/wallets", groupHandler.RemoveWallets)
			}

			// Platform accounts
			accounts := protected.Group("/accounts")
			{
				accountHandler := handlers.NewAccountHandler(s.services)
				accounts.GET("", accountHandler.List)
				accounts.POST("", accountHandler.Create)
				accounts.GET("/:id", accountHandler.Get)
				accounts.PUT("/:id", accountHandler.Update)
				accounts.DELETE("/:id", accountHandler.Delete)
				accounts.GET("/:id/activities", accountHandler.GetActivities)
				accounts.POST("/:id/link-wallet", accountHandler.LinkWallet)
				accounts.POST("/:id/sync", accountHandler.Sync)
			}

			// Campaigns
			campaigns := protected.Group("/campaigns")
			{
				campaignHandler := handlers.NewCampaignHandler(s.services)
				campaigns.GET("", campaignHandler.List)
				campaigns.POST("", campaignHandler.Create)
				campaigns.GET("/:id", campaignHandler.Get)
				campaigns.PUT("/:id", campaignHandler.Update)
				campaigns.DELETE("/:id", campaignHandler.Delete)
				campaigns.GET("/:id/tasks", campaignHandler.GetTasks)
				campaigns.POST("/:id/tasks", campaignHandler.AddTask)
				campaigns.POST("/:id/execute", campaignHandler.ExecuteBulk)
				campaigns.GET("/:id/progress", campaignHandler.GetProgress)
			}

			// Tasks
			tasks := protected.Group("/tasks")
			{
				taskHandler := handlers.NewTaskHandler(s.services)
				tasks.GET("/:id", taskHandler.Get)
				tasks.PUT("/:id", taskHandler.Update)
				tasks.POST("/:id/execute", taskHandler.Execute)
				tasks.POST("/:id/continue", taskHandler.Continue)
				tasks.GET("/:id/executions", taskHandler.GetExecutions)
			}

			// Browser sessions
			browser := protected.Group("/browser")
			{
				browserHandler := handlers.NewBrowserHandler(s.services)
				browser.GET("/profiles", browserHandler.ListProfiles)
				browser.POST("/profiles", browserHandler.CreateProfile)
				browser.DELETE("/profiles/:id", browserHandler.DeleteProfile)
				browser.POST("/sessions", browserHandler.StartSession)
				browser.GET("/sessions", browserHandler.ListSessions)
				browser.GET("/sessions/:id", browserHandler.GetSession)
				browser.DELETE("/sessions/:id", browserHandler.StopSession)
				browser.POST("/sessions/:id/action", browserHandler.ExecuteAction)
				browser.POST("/sessions/:id/continue", browserHandler.ContinueTask)
				browser.GET("/sessions/:id/screenshot", browserHandler.GetScreenshot)
			}

			// AI Content
			content := protected.Group("/content")
			{
				contentHandler := handlers.NewContentHandler(s.services)
				content.POST("/generate", contentHandler.Generate)
				content.GET("/drafts", contentHandler.ListDrafts)
				content.GET("/drafts/:id", contentHandler.GetDraft)
				content.PUT("/drafts/:id", contentHandler.UpdateDraft)
				content.DELETE("/drafts/:id", contentHandler.DeleteDraft)
				content.POST("/drafts/:id/approve", contentHandler.ApproveDraft)
				content.POST("/schedule", contentHandler.Schedule)
				content.GET("/scheduled", contentHandler.ListScheduled)
				content.DELETE("/scheduled/:id", contentHandler.CancelScheduled)
			}

			// Automation jobs
			jobs := protected.Group("/jobs")
			{
				jobHandler := handlers.NewJobHandler(s.services)
				jobs.GET("", jobHandler.List)
				jobs.POST("", jobHandler.Create)
				jobs.GET("/:id", jobHandler.Get)
				jobs.PUT("/:id", jobHandler.Update)
				jobs.DELETE("/:id", jobHandler.Delete)
				jobs.POST("/:id/start", jobHandler.Start)
				jobs.POST("/:id/stop", jobHandler.Stop)
				jobs.GET("/:id/logs", jobHandler.GetLogs)
			}

			// Proxy management
			proxies := protected.Group("/proxies")
			{
				proxyHandler := handlers.NewProxyHandler(s.services)
				proxies.GET("", proxyHandler.List)
				proxies.POST("", proxyHandler.Create)
				proxies.PUT("/:id", proxyHandler.Update)
				proxies.DELETE("/:id", proxyHandler.Delete)
				proxies.POST("/:id/test", proxyHandler.Test)
				proxies.POST("/bulk", proxyHandler.BulkCreate)
			}

			// Dashboard stats
			dashboard := protected.Group("/dashboard")
			{
				dashboardHandler := handlers.NewDashboardHandler(s.services)
				dashboard.GET("/stats", dashboardHandler.GetStats)
				dashboard.GET("/activity", dashboardHandler.GetRecentActivity)
				dashboard.GET("/campaigns/active", dashboardHandler.GetActiveCampaigns)
			}
		}

		// WebSocket endpoint
		v1.GET("/ws", func(c *gin.Context) {
			websocket.ServeWs(s.wsHub, c.Writer, c.Request, s.config.JWTSecret)
		})
	}
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
