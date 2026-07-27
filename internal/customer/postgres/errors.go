package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// pgUniqueViolation is the Postgres error code for a unique constraint
// violation. See:
// https://www.postgresql.org/docs/current/errcodes-appendix.html
//
// Hardcoded rather than pulled from a constants package (e.g.
// github.com/jackc/pgerrcode) for the same reason as
// internal/inventory/postgres/errors.go: avoid a dependency for one
// stable string.
const pgUniqueViolation = "23505"

// translateError maps a lower-level error into a platform apperror so
// PostgreSQL- and pgx-specific error types never leak past this package.
// op names the operation that failed, for context in the wrapped message.
//
// This mirrors internal/auth/postgres/errors.go's translateError, which
// mirrors internal/inventory/postgres/errors.go's: there is no
// foreign-key-violation branch here, because the customers table has no
// foreign keys (goal scope: "no services, no equipment relationships" —
// nothing yet references a Customer, and a Customer references nothing).
// Handling an error code that can never occur against this schema would
// be dead code, not defensive programming.
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
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return apperror.Conflict(fmt.Sprintf("%s: already exists", op))
	}

	return apperror.Internal(op, err)
}
