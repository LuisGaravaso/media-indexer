package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/LuisGaravaso/media-indexer/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPool_EmptyURL(t *testing.T) {
	ctx := context.Background()
	pool, err := database.NewPool(ctx, "")
	require.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "database URL cannot be empty")
}

func TestNewPool_InvalidURL(t *testing.T) {
	ctx := context.Background()
	pool, err := database.NewPool(ctx, "invalid-postgres-url://foo")
	require.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "failed to parse database configuration")
}

func TestNewPool_ValidConfigCreation(t *testing.T) {
	ctx := context.Background()
	customCfg := database.PoolConfig{
		MaxConns:        10,
		MinConns:        1,
		MaxConnLifetime: 2 * time.Hour,
		MaxConnIdleTime: 10 * time.Minute,
	}

	pool, err := database.NewPool(ctx, "postgres://user:pass@localhost:5432/testdb?sslmode=disable", customCfg)
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer pool.Close()

	assert.Equal(t, int32(10), pool.Config().MaxConns)
	assert.Equal(t, int32(1), pool.Config().MinConns)
	assert.Equal(t, 2*time.Hour, pool.Config().MaxConnLifetime)
	assert.Equal(t, 10*time.Minute, pool.Config().MaxConnIdleTime)
}

func TestDefaultPoolConfig(t *testing.T) {
	cfg := database.DefaultPoolConfig()
	assert.Equal(t, int32(25), cfg.MaxConns)
	assert.Equal(t, int32(2), cfg.MinConns)
	assert.Equal(t, time.Hour, cfg.MaxConnLifetime)
	assert.Equal(t, 30*time.Minute, cfg.MaxConnIdleTime)
}
