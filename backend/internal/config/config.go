package config

import (
	"os"
)

type Config struct {
	// Database
	DatabaseURL string
	RedisURL    string

	// Security
	JWTSecret     string
	EncryptionKey string
	CORSOrigin    string

	// Internal Services
	AIServiceURL string
	BrowserWSURL string

	// Platform API Keys
	NeynarAPIKey        string // Farcaster via Neynar
	FarcasterAPIKey     string // Legacy
	TelegramBotToken    string
	TwitterAPIKey       string
	TwitterSecret       string
	TwitterBearerToken  string
	TwitterAccessToken  string
	TwitterAccessSecret string

	// AI
	OpenAIKey string

	// Blockchain RPC URLs
	EthereumRPCURL string
	SolanaRPCURL   string

	// Blockchain Explorer APIs
	BlockchairAPIKey string

	// Storage
	ProofStoragePath string // Path for storing proof screenshots
}

func Load() *Config {
	return &Config{
		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),

		// Security
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "32-byte-key-for-wallet-encryption"),
		CORSOrigin:    getEnv("CORS_ORIGIN", "*"),

		// Internal Services
		AIServiceURL: getEnv("AI_SERVICE_URL", "http://localhost:8001"),
		BrowserWSURL: getEnv("BROWSER_WS_URL", "ws://localhost:9222"),

		// Platform API Keys
		NeynarAPIKey:        getEnv("NEYNAR_API_KEY", ""),
		FarcasterAPIKey:     getEnv("FARCASTER_API_KEY", ""),
		TelegramBotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		TwitterAPIKey:       getEnv("TWITTER_API_KEY", ""),
		TwitterSecret:       getEnv("TWITTER_API_SECRET", ""),
		TwitterBearerToken:  getEnv("TWITTER_BEARER_TOKEN", ""),
		TwitterAccessToken:  getEnv("TWITTER_ACCESS_TOKEN", ""),
		TwitterAccessSecret: getEnv("TWITTER_ACCESS_SECRET", ""),

		// AI
		OpenAIKey: getEnv("OPENAI_API_KEY", ""),

		// Blockchain RPC URLs
		EthereumRPCURL: getEnv("ETHEREUM_RPC_URL", "https://eth.llamarpc.com"),
		SolanaRPCURL:   getEnv("SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com"),

		// Blockchain Explorer APIs
		BlockchairAPIKey: getEnv("BLOCKCHAIR_API_KEY", "G___21MVuo36XwaAt1fKa5j4rrB9gyKE"),

		// Storage
		ProofStoragePath: getEnv("PROOF_STORAGE_PATH", "./storage/proofs"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
