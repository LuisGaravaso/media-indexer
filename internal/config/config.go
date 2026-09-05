package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config stores application configuration values for media-indexer.
type Config struct {
	// Server
	Port    string
	GinMode string

	// Database
	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration

	// Supabase & Auth
	SupabaseURL       string
	SupabaseJWTSecret string

	// GCP & Events
	GCPProjectID       string
	PubSubSubscription string
	StorageBucket      string

	// AI & Models
	GeminiAPIKey      string
	GeminiModel       string
	EmbeddingModel    string
	WorkerConcurrency int
}

// Load reads configuration from .env file (if present) and environment variables.
func Load() *Config {
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	ginMode := getEnv("GIN_MODE", "release")

	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgrespassword@localhost:5432/media_manager_dev?sslmode=disable")
	dbMaxConns := int32(getEnvAsInt("DB_MAX_CONNS", 25))
	dbMinConns := int32(getEnvAsInt("DB_MIN_CONNS", 2))
	dbMaxConnLifetime := getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Hour)
	dbMaxConnIdleTime := getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute)

	supabaseURL := getEnv("SUPABASE_URL", "")
	supabaseJWTSecret := getEnv("SUPABASE_JWT_SECRET", "")

	gcpProjectID := getEnv("GCP_PROJECT_ID", "reeler-sandbox")
	pubsubSubscription := getEnv("PUBSUB_SUBSCRIPTION", "media-indexer-sub")
	storageBucket := getEnv("STORAGE_BUCKET", "reeler-media-sandbox")

	geminiAPIKey := getEnv("GEMINI_API_KEY", "")
	geminiModel := getEnv("GEMINI_MODEL", "gemini-3.6-flash")
	embeddingModel := getEnv("EMBEDDING_MODEL", "text-embedding-004")
	workerConcurrency := getEnvAsInt("WORKER_CONCURRENCY", 5)

	return &Config{
		Port:               port,
		GinMode:            ginMode,
		DatabaseURL:        dbURL,
		DBMaxConns:         dbMaxConns,
		DBMinConns:         dbMinConns,
		DBMaxConnLifetime:  dbMaxConnLifetime,
		DBMaxConnIdleTime:  dbMaxConnIdleTime,
		SupabaseURL:        supabaseURL,
		SupabaseJWTSecret:  supabaseJWTSecret,
		GCPProjectID:       gcpProjectID,
		PubSubSubscription: pubsubSubscription,
		StorageBucket:      storageBucket,
		GeminiAPIKey:       geminiAPIKey,
		GeminiModel:        geminiModel,
		EmbeddingModel:     embeddingModel,
		WorkerConcurrency: workerConcurrency,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
