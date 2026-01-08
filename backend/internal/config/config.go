package config

import (
	"os"
)

type Config struct {
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	AIServiceURL    string
	EncryptionKey   string
	BrowserWSURL    string
	FarcasterAPIKey string
	TwitterAPIKey   string
	TwitterSecret   string
	OpenAIKey       string
}

func Load() *Config {
	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:       getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		AIServiceURL:    getEnv("AI_SERVICE_URL", "http://localhost:8001"),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", "32-byte-key-for-wallet-encryption"),
		BrowserWSURL:    getEnv("BROWSER_WS_URL", "ws://localhost:9222"),
		FarcasterAPIKey: getEnv("FARCASTER_API_KEY", ""),
		TwitterAPIKey:   getEnv("TWITTER_API_KEY", ""),
		TwitterSecret:   getEnv("TWITTER_API_SECRET", ""),
		OpenAIKey:       getEnv("OPENAI_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
