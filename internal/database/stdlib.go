package database

import (
	"database/sql"
	"fmt"

	// Registers the "pgx" database/sql driver.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// OpenStdlib opens a database/sql handle backed by the pgx driver. It exists
// solely for tooling that requires the standard library interface — today,
// only the goose migration runner, since goose's public API predates pgx's
// native interface. Application code must use Connect and the Querier
// interface instead; this is the one deliberate exception to the "pgx, not
// database/sql" rule, and it is confined to this function.
func OpenStdlib(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: open stdlib handle: %w", err)
	}
	return db, nil
}
