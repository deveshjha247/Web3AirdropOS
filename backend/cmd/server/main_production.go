//go:build production
// +build production

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"github.com/web3airdropos/backend/internal/api"
	"github.com/web3airdropos/backend/internal/audit"
	"github.com/web3airdropos/backend/internal/auth"
	"github.com/web3airdropos/backend/internal/config"
	"github.com/web3airdropos/backend/internal/database"
	"github.com/web3airdropos/backend/internal/jobs"
	"github.com/web3airdropos/backend/internal/locks"
	"github.com/web3airdropos/backend/internal/queue"
	"github.com/web3airdropos/backend/internal/tasks"
	"github.com/web3airdropos/backend/internal/vault"
	"github.com/web3airdropos/backend/internal/websocket"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	log.Println("🚀 Starting Web3AirdropOS Backend...")

	// Initialize configuration
	cfg := config.Load()
	validateConfig(cfg)

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	log.Println("✅ Database connected")

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}
	log.Println("✅ Migrations completed")

	// Initialize Redis
	redisClient := database.ConnectRedis(cfg.RedisURL)
	if redisClient != nil {
		log.Println("✅ Redis connected")
	}

	// Initialize production components

	// 1. Audit Logger
	auditLogger := audit.NewLogger(db)
	log.Println("✅ Audit logger initialized")

	// 2. Secrets Vault
	secretsVault, err := vault.NewVault(db, vault.Config{
		MasterKey: cfg.EncryptionKey,
	})
	if err != nil {
		log.Fatalf("❌ Failed to initialize vault: %v", err)
	}
	log.Println("✅ Secrets vault initialized")

	// 3. Distributed Lock Manager
	lockManager := locks.NewLockManager(redisClient)
	log.Println("✅ Lock manager initialized")

	// 4. Rate Limiter
	rateLimiter := auth.NewRateLimiter(redisClient)
	log.Println("✅ Rate limiter initialized")

	// 5. Auth Service
	authService := auth.NewAuthService(db, cfg.JWTSecret)
	log.Println("✅ Auth service initialized")

	// 6. Task Queue
	taskQueue := queue.NewQueue(redisClient, "tasks")
	log.Println("✅ Task queue initialized")

	// 7. Task Manager
	taskManager := tasks.NewTaskManager(db, lockManager, taskQueue)
	log.Println("✅ Task manager initialized")

	// 8. WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Println("✅ WebSocket hub started")

	// 9. Job Scheduler
	scheduler := jobs.NewScheduler(db, redisClient, wsHub, cfg)
	go scheduler.Start()
	log.Println("✅ Job scheduler started")

	// 10. Queue Worker
	worker := queue.NewWorker(taskQueue, "main-worker", queue.DefaultWorkerConfig())
	registerQueueHandlers(worker, taskManager, auditLogger)
	go worker.Start(context.Background())
	log.Println("✅ Queue worker started")

	// Create production container with all services
	prodContainer := &api.ProductionContainer{
		Config:      cfg,
		DB:          db,
		Redis:       redisClient,
		WSHub:       wsHub,
		AuditLogger: auditLogger,
		Vault:       secretsVault,
		LockManager: lockManager,
		RateLimiter: rateLimiter,
		AuthService: authService,
		TaskQueue:   taskQueue,
		TaskManager: taskManager,
	}

	// Initialize and start API server
	server := api.NewProductionServer(prodContainer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Graceful shutdown handling
	go func() {
		log.Printf("🚀 Web3AirdropOS Backend running on port %s", port)
		if err := server.Run(":" + port); err != nil {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down gracefully...")

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop worker
	worker.Stop()

	// Stop audit logger
	auditLogger.Stop()

	// Stop scheduler (if it has a Stop method)
	// scheduler.Stop()

	// Cleanup expired tokens
	if deleted, err := authService.CleanupExpiredTokens(ctx); err == nil {
		log.Printf("🧹 Cleaned up %d expired tokens", deleted)
	}

	log.Println("✅ Shutdown complete")
}

func validateConfig(cfg *config.Config) {
	if cfg.JWTSecret == "" || cfg.JWTSecret == "your-secret-key-change-in-production" {
		log.Println("⚠️  WARNING: Using default JWT secret. Set JWT_SECRET in production!")
	}

	if cfg.EncryptionKey == "" || cfg.EncryptionKey == "32-byte-key-for-wallet-encryption" {
		log.Println("⚠️  WARNING: Using default encryption key. Set ENCRYPTION_KEY in production!")
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("❌ DATABASE_URL is required")
	}
}

func registerQueueHandlers(worker *queue.Worker, taskManager *tasks.TaskManager, auditLogger *audit.Logger) {
	// Task retry handler
	worker.RegisterHandler("task_retry", func(ctx context.Context, job *queue.Job) error {
		var payload struct {
			ExecutionID string `json:"execution_id"`
			TaskID      string `json:"task_id"`
		}

		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}

		execID, err := uuid.Parse(payload.ExecutionID)
		if err != nil {
			return err
		}

		_, err = taskManager.RetryExecution(ctx, execID)
		return err
	})

	// Audit log cleanup handler
	worker.RegisterHandler("audit_cleanup", func(ctx context.Context, job *queue.Job) error {
		deleted, err := auditLogger.Cleanup(ctx, 90) // 90 days retention
		if err != nil {
			return err
		}
		log.Printf("🧹 Cleaned up %d old audit logs", deleted)
		return nil
	})
}
