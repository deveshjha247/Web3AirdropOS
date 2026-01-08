package database

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/web3airdropos/backend/internal/models"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("✅ Database connected successfully")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")
	
	err := db.AutoMigrate(
		// Core models
		&models.User{},
		
		// Wallet models
		&models.Wallet{},
		&models.WalletGroup{},
		&models.Transaction{},
		
		// Platform account models
		&models.PlatformAccount{},
		&models.AccountActivity{},
		
		// Campaign models
		&models.Campaign{},
		&models.CampaignTask{},
		&models.TaskExecution{},
		
		// Automation models
		&models.AutomationJob{},
		&models.JobLog{},
		
		// Content models
		&models.ContentDraft{},
		&models.ScheduledPost{},
		
		// Browser session models
		&models.BrowserSession{},
		&models.BrowserProfile{},
	)
	
	if err != nil {
		return err
	}

	log.Println("✅ Migrations completed successfully")
	return nil
}

func ConnectRedis(redisURL string) *redis.Client {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("⚠️ Invalid Redis URL, using default: %v", err)
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opt)
	
	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis connection failed: %v", err)
	} else {
		log.Println("✅ Redis connected successfully")
	}

	return client
}
