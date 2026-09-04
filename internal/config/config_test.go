package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/LuisGaravaso/media-indexer/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// Clean environment variables
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("GIN_MODE")
	_ = os.Unsetenv("DATABASE_URL")
	_ = os.Unsetenv("GCP_PROJECT_ID")
	_ = os.Unsetenv("PUBSUB_SUBSCRIPTION")
	_ = os.Unsetenv("STORAGE_BUCKET")
	_ = os.Unsetenv("GEMINI_MODEL")
	_ = os.Unsetenv("EMBEDDING_MODEL")
	_ = os.Unsetenv("WORKER_CONCURRENCY")

	cfg := config.Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "release", cfg.GinMode)
	assert.Equal(t, "reeler-sandbox", cfg.GCPProjectID)
	assert.Equal(t, "media-indexer-sub", cfg.PubSubSubscription)
	assert.Equal(t, "reeler-media-sandbox", cfg.StorageBucket)
	assert.Equal(t, "gemini-2.0-flash", cfg.GeminiModel)
	assert.Equal(t, "text-embedding-004", cfg.EmbeddingModel)
	assert.Equal(t, 5, cfg.WorkerConcurrency)
	assert.Equal(t, int32(25), cfg.DBMaxConns)
	assert.Equal(t, int32(2), cfg.DBMinConns)
	assert.Equal(t, time.Hour, cfg.DBMaxConnLifetime)
	assert.Equal(t, 30*time.Minute, cfg.DBMaxConnIdleTime)
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("GIN_MODE", "debug")
	t.Setenv("GCP_PROJECT_ID", "reeler-prod")
	t.Setenv("PUBSUB_SUBSCRIPTION", "media-indexer-prod-sub")
	t.Setenv("STORAGE_BUCKET", "reeler-media-prod")
	t.Setenv("GEMINI_MODEL", "gemini-1.5-pro")
	t.Setenv("EMBEDDING_MODEL", "text-embedding-005")
	t.Setenv("WORKER_CONCURRENCY", "10")
	t.Setenv("DB_MAX_CONNS", "50")
	t.Setenv("DB_MIN_CONNS", "5")
	t.Setenv("DB_MAX_CONN_LIFETIME", "2h")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "15m")

	cfg := config.Load()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "debug", cfg.GinMode)
	assert.Equal(t, "reeler-prod", cfg.GCPProjectID)
	assert.Equal(t, "media-indexer-prod-sub", cfg.PubSubSubscription)
	assert.Equal(t, "reeler-media-prod", cfg.StorageBucket)
	assert.Equal(t, "gemini-1.5-pro", cfg.GeminiModel)
	assert.Equal(t, "text-embedding-005", cfg.EmbeddingModel)
	assert.Equal(t, 10, cfg.WorkerConcurrency)
	assert.Equal(t, int32(50), cfg.DBMaxConns)
	assert.Equal(t, int32(5), cfg.DBMinConns)
	assert.Equal(t, 2*time.Hour, cfg.DBMaxConnLifetime)
	assert.Equal(t, 15*time.Minute, cfg.DBMaxConnIdleTime)
}

func TestLoad_InvalidNumericFallbacks(t *testing.T) {
	t.Setenv("WORKER_CONCURRENCY", "not_a_number")
	t.Setenv("DB_MAX_CONN_LIFETIME", "invalid_duration")

	cfg := config.Load()

	assert.Equal(t, 5, cfg.WorkerConcurrency)
	assert.Equal(t, time.Hour, cfg.DBMaxConnLifetime)
}
