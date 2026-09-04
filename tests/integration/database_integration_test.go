//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/LuisGaravaso/media-indexer/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresDatabase_Integration(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/media_indexer_dev?sslmode=disable"
	}

	pool, err := database.NewPool(ctx, dbURL)
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer pool.Close()

	// 1. Verify live ping
	pingCtx, pingCancel := context.WithTimeout(ctx, 2*time.Second)
	defer pingCancel()
	err = pool.Ping(pingCtx)
	require.NoError(t, err)

	// 2. Verify pgvector extension
	var vectorExtCount int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM pg_extension WHERE extname = 'vector'").Scan(&vectorExtCount)
	require.NoError(t, err)
	assert.Equal(t, 1, vectorExtCount, "pgvector extension must be enabled")

	// 3. Verify media_semantics and media_scenes tables exist
	var semanticsCount int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_name = 'media_semantics'").Scan(&semanticsCount)
	require.NoError(t, err)
	assert.Equal(t, 1, semanticsCount, "media_semantics table must exist")

	var scenesCount int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_name = 'media_scenes'").Scan(&scenesCount)
	require.NoError(t, err)
	assert.Equal(t, 1, scenesCount, "media_scenes table must exist")
}
