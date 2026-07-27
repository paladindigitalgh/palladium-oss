// Package migrate runs SQL schema migrations via goose. It is invoked by
// cmd/migrate, a binary separate from the API server: schema changes are an
// explicit, auditable deploy step, not something the server does implicitly
// on boot (per CLAUDE.md: "Every schema change uses migrations. Never
// modify production data directly.").
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// Runner applies and inspects SQL schema migrations.
type Runner struct {
	provider *goose.Provider
}

// New builds a Runner that reads migrations from fsys and tracks applied
// versions in db.
//
// db must be a *sql.DB backed by a Postgres driver (see
// internal/database.OpenStdlib) — goose's public API predates pgx's native
// interface and is database/sql based, which is the one place this
// application steps outside pgx for Postgres access.
func New(db *sql.DB, fsys fs.FS) (*Runner, error) {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys)
	if err != nil {
		return nil, fmt.Errorf("migrate: new provider: %w", err)
	}
	return &Runner{provider: provider}, nil
}

// Up applies every pending migration and returns how many were run.
func (r *Runner) Up(ctx context.Context) (int, error) {
	results, err := r.provider.Up(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrate: up: %w", err)
	}
	return len(results), nil
}

// Version reports the schema version currently recorded in the database.
func (r *Runner) Version(ctx context.Context) (int64, error) {
	version, err := r.provider.GetDBVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrate: version: %w", err)
	}
	return version, nil
}

// HasPending reports whether any migration has not yet been applied.
func (r *Runner) HasPending(ctx context.Context) (bool, error) {
	pending, err := r.provider.HasPending(ctx)
	if err != nil {
		return false, fmt.Errorf("migrate: has pending: %w", err)
	}
	return pending, nil
}
