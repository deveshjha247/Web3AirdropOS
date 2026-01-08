package main

import (
	"log"
	"os"

	"github.com/web3airdropos/backend/internal/api"
	"github.com/web3airdropos/backend/internal/config"
	"github.com/web3airdropos/backend/internal/database"
	"github.com/web3airdropos/backend/internal/jobs"
	"github.com/web3airdropos/backend/internal/websocket"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis
	redisClient := database.ConnectRedis(cfg.RedisURL)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize job scheduler
	scheduler := jobs.NewScheduler(db, redisClient, wsHub)
	go scheduler.Start()

	// Initialize and start API server
	server := api.NewServer(cfg, db, redisClient, wsHub)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Web3AirdropOS Backend starting on port %s", port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
