package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/web3airdropos/backend/internal/api/handlers"
	"github.com/web3airdropos/backend/internal/audit"
	"github.com/web3airdropos/backend/internal/auth"
	"github.com/web3airdropos/backend/internal/config"
	"github.com/web3airdropos/backend/internal/locks"
	"github.com/web3airdropos/backend/internal/queue"
	"github.com/web3airdropos/backend/internal/services"
	"github.com/web3airdropos/backend/internal/tasks"
	"github.com/web3airdropos/backend/internal/vault"
	"github.com/web3airdropos/backend/internal/websocket"
)

// ProductionContainer holds all production service instances
type ProductionContainer struct {
	Config      *config.Config
	DB          *gorm.DB
	Redis       *redis.Client
	WSHub       *websocket.Hub
	AuditLogger *audit.Logger
	Vault       *vault.Vault
	LockManager *locks.LockManager
	RateLimiter *auth.RateLimiter
	AuthService *auth.AuthService
	TaskQueue   *queue.Queue
	TaskManager *tasks.TaskManager
}

// ProductionServer is the production-ready API server
type ProductionServer struct {
	router    *gin.Engine
	container *ProductionContainer
	services  *services.Container
}

// NewProductionServer creates a new production-ready server
func NewProductionServer(container *ProductionContainer) *ProductionServer {
	// Set Gin to release mode in production
	gin.SetMode(gin.ReleaseMode)

	// Initialize services container for handlers
	svc := services.NewContainer(
		container.Config,
		container.DB,
		container.Redis,
		container.WSHub,
	)

	server := &ProductionServer{
		router:    gin.New(),
		container: container,
		services:  svc,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// setupMiddleware configures all middleware
func (s *ProductionServer) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request logging (structured)
	s.router.Use(s.requestLogger())

	// CORS
	s.router.Use(s.corsMiddleware())

	// Security headers
	s.router.Use(s.securityHeaders())

	// Global rate limiting (IP-based)
	s.router.Use(s.globalRateLimit())

	// Audit middleware
	s.router.Use(auth.AuditMiddleware())
}

// requestLogger logs requests in structured format
func (s *ProductionServer) requestLogger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	})
}

// corsMiddleware handles CORS
func (s *ProductionServer) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*") // Configure for production
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// securityHeaders adds security headers
func (s *ProductionServer) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// globalRateLimit applies global rate limiting
func (s *ProductionServer) globalRateLimit() gin.HandlerFunc {
	return auth.RateLimitMiddleware(s.container.RateLimiter, auth.RateLimitDefault)
}

// authRateLimit applies stricter rate limiting for auth endpoints
func (s *ProductionServer) authRateLimit() gin.HandlerFunc {
	return auth.RateLimitMiddleware(s.container.RateLimiter, auth.RateLimitAuth)
}

// writeRateLimit applies rate limiting for write operations
func (s *ProductionServer) writeRateLimit() gin.HandlerFunc {
	return auth.RateLimitMiddleware(s.container.RateLimiter, auth.RateLimitWrite)
}

// authRequired requires authentication
func (s *ProductionServer) authRequired() gin.HandlerFunc {
	return auth.AuthMiddleware(s.container.AuthService)
}

// setupRoutes configures all API routes
func (s *ProductionServer) setupRoutes() {
	// Health check (public)
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "web3airdropos-backend",
			"version": "1.0.0",
		})
	})

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Auth routes (public, strict rate limit)
		auth := v1.Group("/auth")
		auth.Use(s.authRateLimit())
		{
			authHandler := handlers.NewAuthHandler(s.services)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(s.authRequired())
		{
			// Account logout
			protected.POST("/auth/logout", func(c *gin.Context) {
				// Handle logout
				c.JSON(http.StatusOK, gin.H{"message": "logged out"})
			})

			// Wallet routes
			wallets := protected.Group("/wallets")
			{
				walletHandler := handlers.NewWalletHandler(s.services)
				wallets.GET("", walletHandler.List)
				wallets.POST("", s.writeRateLimit(), walletHandler.Create)
				wallets.GET("/:id", walletHandler.Get)
				wallets.PUT("/:id", s.writeRateLimit(), walletHandler.Update)
				wallets.DELETE("/:id", s.writeRateLimit(), walletHandler.Delete)
				wallets.GET("/:id/balance", walletHandler.GetBalance)
				wallets.GET("/:id/transactions", walletHandler.GetTransactions)
				wallets.POST("/:id/prepare-tx", s.writeRateLimit(), walletHandler.PrepareTransaction)
				wallets.POST("/import", s.writeRateLimit(), walletHandler.Import)
				wallets.POST("/bulk", s.writeRateLimit(), walletHandler.BulkCreate)
			}

			// Wallet groups
			groups := protected.Group("/wallet-groups")
			{
				groupHandler := handlers.NewWalletGroupHandler(s.services)
				groups.GET("", groupHandler.List)
				groups.POST("", s.writeRateLimit(), groupHandler.Create)
				groups.PUT("/:id", s.writeRateLimit(), groupHandler.Update)
				groups.DELETE("/:id", s.writeRateLimit(), groupHandler.Delete)
				groups.POST("/:id/wallets", s.writeRateLimit(), groupHandler.AddWallets)
				groups.DELETE("/:id/wallets", s.writeRateLimit(), groupHandler.RemoveWallets)
			}

			// Platform accounts
			accounts := protected.Group("/accounts")
			{
				accountHandler := handlers.NewAccountHandler(s.services)
				accounts.GET("", accountHandler.List)
				accounts.POST("", s.writeRateLimit(), accountHandler.Create)
				accounts.GET("/:id", accountHandler.Get)
				accounts.PUT("/:id", s.writeRateLimit(), accountHandler.Update)
				accounts.DELETE("/:id", s.writeRateLimit(), accountHandler.Delete)
				accounts.GET("/:id/activities", accountHandler.GetActivities)
				accounts.POST("/:id/link-wallet", s.writeRateLimit(), accountHandler.LinkWallet)
				accounts.POST("/:id/sync", s.writeRateLimit(), accountHandler.Sync)
			}

			// Campaigns
			campaigns := protected.Group("/campaigns")
			{
				campaignHandler := handlers.NewCampaignHandler(s.services)
				campaigns.GET("", campaignHandler.List)
				campaigns.POST("", s.writeRateLimit(), campaignHandler.Create)
				campaigns.GET("/:id", campaignHandler.Get)
				campaigns.PUT("/:id", s.writeRateLimit(), campaignHandler.Update)
				campaigns.DELETE("/:id", s.writeRateLimit(), campaignHandler.Delete)
				campaigns.GET("/:id/tasks", campaignHandler.GetTasks)
				campaigns.POST("/:id/tasks", s.writeRateLimit(), campaignHandler.AddTask)
				campaigns.POST("/:id/execute", s.writeRateLimit(), campaignHandler.ExecuteBulk)
				campaigns.GET("/:id/progress", campaignHandler.GetProgress)
			}

			// Tasks
			tasks := protected.Group("/tasks")
			{
				taskHandler := handlers.NewTaskHandler(s.services)
				tasks.GET("/:id", taskHandler.Get)
				tasks.PUT("/:id", s.writeRateLimit(), taskHandler.Update)
				tasks.POST("/:id/execute", s.writeRateLimit(), taskHandler.Execute)
				tasks.POST("/:id/continue", s.writeRateLimit(), taskHandler.Continue)
				tasks.GET("/:id/executions", taskHandler.GetExecutions)
			}

			// Browser sessions
			browser := protected.Group("/browser")
			{
				browserHandler := handlers.NewBrowserHandler(s.services)
				browser.GET("/profiles", browserHandler.ListProfiles)
				browser.POST("/profiles", s.writeRateLimit(), browserHandler.CreateProfile)
				browser.DELETE("/profiles/:id", s.writeRateLimit(), browserHandler.DeleteProfile)
				browser.POST("/sessions", s.writeRateLimit(), browserHandler.StartSession)
				browser.GET("/sessions", browserHandler.ListSessions)
				browser.GET("/sessions/:id", browserHandler.GetSession)
				browser.DELETE("/sessions/:id", s.writeRateLimit(), browserHandler.StopSession)
				browser.POST("/sessions/:id/action", s.writeRateLimit(), browserHandler.ExecuteAction)
				browser.POST("/sessions/:id/continue", s.writeRateLimit(), browserHandler.ContinueTask)
				browser.GET("/sessions/:id/screenshot", browserHandler.GetScreenshot)
			}

			// AI Content
			content := protected.Group("/content")
			{
				contentHandler := handlers.NewContentHandler(s.services)
				content.POST("/generate", s.writeRateLimit(), contentHandler.Generate)
				content.GET("/drafts", contentHandler.ListDrafts)
				content.GET("/drafts/:id", contentHandler.GetDraft)
				content.PUT("/drafts/:id", s.writeRateLimit(), contentHandler.UpdateDraft)
				content.DELETE("/drafts/:id", s.writeRateLimit(), contentHandler.DeleteDraft)
				content.POST("/drafts/:id/approve", s.writeRateLimit(), contentHandler.ApproveDraft)
				content.POST("/schedule", s.writeRateLimit(), contentHandler.Schedule)
				content.GET("/scheduled", contentHandler.ListScheduled)
				content.DELETE("/scheduled/:id", s.writeRateLimit(), contentHandler.CancelScheduled)
			}

			// Automation jobs
			jobs := protected.Group("/jobs")
			{
				jobHandler := handlers.NewJobHandler(s.services)
				jobs.GET("", jobHandler.List)
				jobs.POST("", s.writeRateLimit(), jobHandler.Create)
				jobs.GET("/:id", jobHandler.Get)
				jobs.PUT("/:id", s.writeRateLimit(), jobHandler.Update)
				jobs.DELETE("/:id", s.writeRateLimit(), jobHandler.Delete)
				jobs.POST("/:id/start", s.writeRateLimit(), jobHandler.Start)
				jobs.POST("/:id/stop", s.writeRateLimit(), jobHandler.Stop)
				jobs.GET("/:id/logs", jobHandler.GetLogs)
			}

			// Proxy management
			proxies := protected.Group("/proxies")
			{
				proxyHandler := handlers.NewProxyHandler(s.services)
				proxies.GET("", proxyHandler.List)
				proxies.POST("", s.writeRateLimit(), proxyHandler.Create)
				proxies.PUT("/:id", s.writeRateLimit(), proxyHandler.Update)
				proxies.DELETE("/:id", s.writeRateLimit(), proxyHandler.Delete)
				proxies.POST("/:id/test", s.writeRateLimit(), proxyHandler.Test)
				proxies.POST("/bulk", s.writeRateLimit(), proxyHandler.BulkCreate)
			}

			// Dashboard stats
			dashboard := protected.Group("/dashboard")
			{
				dashboardHandler := handlers.NewDashboardHandler(s.services)
				dashboard.GET("/stats", dashboardHandler.GetStats)
				dashboard.GET("/activity", dashboardHandler.GetRecentActivity)
				dashboard.GET("/campaigns/active", dashboardHandler.GetActiveCampaigns)
			}

			// Audit logs
			auditLogs := protected.Group("/audit")
			{
				auditLogs.GET("", s.getAuditLogs())
				auditLogs.GET("/:id", s.getAuditLog())
			}

			// Secrets vault
			secrets := protected.Group("/secrets")
			{
				secrets.GET("", s.listSecrets())
				secrets.POST("", s.writeRateLimit(), s.storeSecret())
				secrets.GET("/:name", s.getSecret())
				secrets.PUT("/:name", s.writeRateLimit(), s.updateSecret())
				secrets.DELETE("/:name", s.writeRateLimit(), s.deleteSecret())
			}
		}
	}

	// WebSocket endpoint
	s.router.GET("/ws", func(c *gin.Context) {
		s.container.WSHub.HandleWebSocket(c.Writer, c.Request)
	})
}

// Run starts the server
func (s *ProductionServer) Run(addr string) error {
	return s.router.Run(addr)
}

// Audit log handlers
func (s *ProductionServer) getAuditLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := auth.GetUserID(c)

		logs, total, err := s.container.AuditLogger.Query(c.Request.Context(), &audit.QueryParams{
			UserID: &userID,
			Limit:  50,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"logs":  logs,
			"total": total,
		})
	}
}

func (s *ProductionServer) getAuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Implementation
		c.JSON(http.StatusOK, gin.H{})
	}
}

// Secrets vault handlers
func (s *ProductionServer) listSecrets() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := auth.GetUserID(c)

		secrets, err := s.container.Vault.List(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"secrets": secrets})
	}
}

func (s *ProductionServer) storeSecret() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := auth.GetUserID(c)

		var req struct {
			Name     string                 `json:"name" binding:"required"`
			Value    string                 `json:"value" binding:"required"`
			KeyType  vault.SecretType       `json:"key_type" binding:"required"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		secret, err := s.container.Vault.Store(c.Request.Context(), userID, req.Name, req.Value, req.KeyType, req.Metadata)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Log audit
		s.container.AuditLogger.Log(c.Request.Context(), &audit.LogEntry{
			UserID:   userID,
			Action:   audit.ActionSecretCreate,
			Result:   audit.ResultSuccess,
			TargetID: secret.Name,
		})

		c.JSON(http.StatusCreated, gin.H{
			"id":       secret.ID,
			"name":     secret.Name,
			"key_type": secret.KeyType,
		})
	}
}

func (s *ProductionServer) getSecret() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := auth.GetUserID(c)
		name := c.Param("name")

		value, err := s.container.Vault.Retrieve(c.Request.Context(), userID, name)
		if err != nil {
			if err == vault.ErrSecretNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Log audit
		s.container.AuditLogger.Log(c.Request.Context(), &audit.LogEntry{
			UserID:   userID,
			Action:   audit.ActionSecretAccess,
			Result:   audit.ResultSuccess,
			TargetID: name,
		})

		c.JSON(http.StatusOK, gin.H{"value": value})
	}
}

func (s *ProductionServer) updateSecret() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := auth.GetUserID(c)
		name := c.Param("name")

		var req struct {
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := s.container.Vault.Update(c.Request.Context(), userID, name, req.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "secret updated"})
	}
}

func (s *ProductionServer) deleteSecret() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := auth.GetUserID(c)
		name := c.Param("name")

		if err := s.container.Vault.Delete(c.Request.Context(), userID, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Log audit
		s.container.AuditLogger.Log(c.Request.Context(), &audit.LogEntry{
			UserID:   userID,
			Action:   audit.ActionSecretDelete,
			Result:   audit.ResultSuccess,
			TargetID: name,
		})

		c.JSON(http.StatusOK, gin.H{"message": "secret deleted"})
	}
}
