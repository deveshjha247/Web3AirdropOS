package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"github.com/web3airdropos/backend/internal/api"
	"github.com/web3airdropos/backend/internal/config"
	"github.com/web3airdropos/backend/internal/database"
	"github.com/web3airdropos/backend/internal/health"
	"github.com/web3airdropos/backend/internal/jobs"
	"github.com/web3airdropos/backend/internal/logger"
	"github.com/web3airdropos/backend/internal/migrations"
	"github.com/web3airdropos/backend/internal/websocket"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		// Not an error in production - use env vars directly
	}

	// Get environment
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	// Initialize structured logger
	logger.Init(logger.Config{
		Level:  getEnvOrDefault("LOG_LEVEL", "info"),
		Pretty: env == "development",
	})

	log := logger.Get()
	log.Info().
		Str("env", env).
		Str("version", "1.0.0").
		Msg("Starting Web3AirdropOS Backend")

	// Load configuration
	cfg := config.Load()
	validateConfig(cfg, log)

	// Connect to PostgreSQL
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	log.Info().Msg("Connected to PostgreSQL")

	// Run migrations on startup (dev mode) or check (production)
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get SQL DB")
	}

	if env == "development" || os.Getenv("RUN_MIGRATIONS") == "true" {
		if err := migrations.Run(sqlDB, "web3airdropos"); err != nil {
			log.Fatal().Err(err).Msg("Failed to run migrations")
		}
	} else {
		version, dirty, err := migrations.Status(sqlDB, "web3airdropos")
		if err != nil {
			log.Warn().Err(err).Msg("Failed to check migration status")
		} else {
			log.Info().Uint("version", version).Bool("dirty", dirty).Msg("Migration status")
		}
	}

	// Connect to Redis
	var redisClient *redis.Client
	if cfg.RedisURL != "" {
		redisClient = database.ConnectRedis(cfg.RedisURL)
		if redisClient != nil {
			log.Info().Msg("Connected to Redis")
		}
	}

	// Initialize health checker
	healthChecker := health.NewChecker(db, redisClient)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Info().Msg("WebSocket hub started")

	// Initialize job scheduler
	scheduler := jobs.NewScheduler(db, redisClient, wsHub)
	go scheduler.Start()
	log.Info().Msg("Job scheduler started")

	// Initialize API server
	server := api.NewServer(cfg, db, redisClient, wsHub)

	// Register health endpoints
	healthChecker.RegisterRoutes(server.Router())

	// Get port
	port := getEnvOrDefault("PORT", "8080")
	addr := ":" + port

	// Create HTTP server
	srv := &http.Server{
		Addr:         addr,
		Handler:      server.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Mark as ready after startup
	healthChecker.SetReady(true)
	log.Info().Msg("Service is ready")

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info().Str("signal", sig.String()).Msg("Shutting down")

	// Mark as not ready (for k8s)
	healthChecker.SetReady(false)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	// Close database
	if err := sqlDB.Close(); err != nil {
		log.Error().Err(err).Msg("Database close error")
	}

	// Close Redis
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("Redis close error")
		}
	}

	log.Info().Msg("Shutdown complete")
}

func validateConfig(cfg *config.Config, log *zerolog.Logger) {
	errors := []string{}

	if cfg.DatabaseURL == "" {
		errors = append(errors, "DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" || cfg.JWTSecret == "your-secret-key-change-in-production" {
		if os.Getenv("ENV") != "development" {
			errors = append(errors, "JWT_SECRET must be set in production")
		} else {
			log.Warn().Msg("Using default JWT_SECRET - NOT SAFE FOR PRODUCTION")
		}
	}

	if cfg.EncryptionKey == "" || cfg.EncryptionKey == "32-byte-key-for-wallet-encryption" {
		if os.Getenv("ENV") != "development" {
			errors = append(errors, "ENCRYPTION_KEY must be set in production")
		} else {
			log.Warn().Msg("Using default ENCRYPTION_KEY - NOT SAFE FOR PRODUCTION")
		}
	}

	if len(errors) > 0 {
		for _, e := range errors {
			log.Error().Msg(e)
		}
		log.Fatal().Msg("Configuration validation failed")
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Import for validateConfig
import "github.com/rs/zerolog"
