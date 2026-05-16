package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a connection pool to PostgreSQL using the provided connection string.
// The pool is configured for use with Supabase transaction pooler (PgBouncer).
func Connect(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	// Pool tuning suitable for Cloud Run (multiple instances, short-lived requests).
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	// Initial connection through Supabase transaction pooler can be slow on first
	// handshake (cold start, cross-region). Allow up to 30s before giving up.
	cfg.ConnConfig.ConnectTimeout = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Verify connectivity early so startup fails fast if DB is unreachable.
	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
