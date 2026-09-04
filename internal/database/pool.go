package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pinger defines the database ping and close contract for health checks and lifecycle.
type Pinger interface {
	Ping(ctx context.Context) error
	Close()
}

// PoolConfig holds database connection pool configuration options.
type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultPoolConfig provides production-ready default pool settings.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:        25,
		MinConns:        2,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// NewPool parses the connection string, applies pool options, and establishes a pgx connection pool.
func NewPool(ctx context.Context, databaseURL string, poolCfg ...PoolConfig) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL cannot be empty")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// Use unnamed extended protocol statements (DescribeExec) for full compatibility with
	// transaction-mode poolers (Supabase / PgBouncer) while correctly inferring parameter types (JSON, UUID, pgvector, etc.)
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec

	cfg := DefaultPoolConfig()
	if len(poolCfg) > 0 {
		cfg = poolCfg[0]
	}

	if cfg.MaxConns > 0 {
		config.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		config.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		config.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		config.MaxConnIdleTime = cfg.MaxConnIdleTime
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return pool, nil
}
