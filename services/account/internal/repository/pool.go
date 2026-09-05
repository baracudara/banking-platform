package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool builds a tuned pgx pool and verifies the connection before
// returning. It takes a DSN and plain numbers rather than the Config type:
// the repository layer stays unaware of how the process is configured, so
// a test can construct a pool without building a Config first.
func NewPool(ctx context.Context, dsn string, minConns, maxConns int32, connectTimeout time.Duration) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	cfg.MinConns = minConns
	cfg.MaxConns = maxConns

	// Recycle connections so that a long-lived pool doesn't pin server-side
	// state forever and so a failed-over Postgres is picked up eventually.
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// pgxpool connects lazily, so without an explicit ping a service with a
	// wrong password starts up "successfully" and only fails on the first
	// real request. Fail fast at boot instead.
	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
