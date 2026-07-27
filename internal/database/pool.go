package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/retry"
)

// Pool wraps a pgx connection pool. It is the concrete type repositories
// receive at construction; they should depend on the narrower Querier and
// Transactor interfaces below rather than *Pool itself wherever practical.
type Pool struct {
	pool *pgxpool.Pool
}

// Connect builds a connection pool from cfg. It does not block on network
// access: pgxpool establishes connections lazily, so Connect only fails for
// an unusable configuration (e.g. an invalid DSN). Use WarmUp to verify
// connectivity, and the health check for ongoing verification.
func Connect(ctx context.Context, cfg Config) (*Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: parse config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: create pool: %w", err)
	}

	return &Pool{pool: pool}, nil
}

// WarmUp pings the database repeatedly according to backoff, giving a
// dependency that starts slightly after this process (a common race with
// container orchestration) a bounded grace period to become reachable. Each
// individual ping is bounded by attemptTimeout (Config.ConnectTimeout);
// the overall number of attempts is capped by maxAttempts. The caller
// decides whether a failure here is fatal — ongoing availability is
// enforced per-request by the readiness check regardless of the outcome.
func (p *Pool) WarmUp(ctx context.Context, backoff retry.Backoff, maxAttempts int, attemptTimeout time.Duration) error {
	return retry.Do(ctx, backoff, maxAttempts, func() error {
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		defer cancel()
		return p.Ping(attemptCtx)
	})
}

// Close releases all pooled connections. It blocks until every connection
// currently in use has been returned.
func (p *Pool) Close() {
	p.pool.Close()
}

// Ping verifies that at least one connection in the pool is usable.
func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Exec implements Querier.
func (p *Pool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

// Query implements Querier.
func (p *Pool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// QueryRow implements Querier.
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// BeginTx implements Transactor.
func (p *Pool) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return p.pool.Begin(ctx)
}

// Stat exposes pool statistics (open/idle connections, etc.) for
// diagnostics and future metrics.
func (p *Pool) Stat() *pgxpool.Stat {
	return p.pool.Stat()
}
