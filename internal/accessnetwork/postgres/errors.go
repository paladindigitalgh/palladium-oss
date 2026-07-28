package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// Postgres error codes this package distinguishes. See:
// https://www.postgresql.org/docs/current/errcodes-appendix.html
//
// Hardcoded rather than pulled from a constants package (e.g.
// github.com/jackc/pgerrcode) for the same reason as
// internal/inventory/postgres/errors.go: avoid a dependency for two
// stable strings.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// translateError maps a lower-level error into a platform apperror so
// PostgreSQL- and pgx-specific error types never leak past this package.
// op names the operation that failed, for context in the wrapped message.
//
// This includes the foreign-key-violation branch even though nothing in
// the access_networks table itself references another table: olts.access_network_id
// references access_networks(id) ON DELETE RESTRICT (see
// internal/olt/postgres and database/migrations/00017_olt_olts.sql), so
// deleting an AccessNetwork that still has OLTs is a real, reachable
// outcome against this schema. This is the same lesson
// internal/catalog/postgres/errors.go's doc comment describes learning
// from internal/customer/postgres/errors.go: catalog's errors.go was
// written with its own eventual child's FK in mind from the start rather
// than waiting to be caught by a failing test, and this package follows
// that same precedent.
//
// Callers check for pgx.ErrNoRows themselves before calling this
// function, for the same reason as every other repository in this
// codebase: "no rows" is an expected outcome translated to
// apperror.NotFound with a message naming what was missing, not a
// failure this generic translator should guess a message for.
func translateError(op string, err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return apperror.Conflict(fmt.Sprintf("%s: already exists", op))
		case pgForeignKeyViolation:
			return apperror.Conflict(fmt.Sprintf("%s: violates a foreign key relationship", op))
		}
	}

	return apperror.Internal(op, err)
}
