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
// This mirrors internal/inventory/postgres/errors.go's translateError —
// including the foreign-key-violation branch, unlike
// internal/customer/postgres/errors.go and internal/auth/postgres/errors.go,
// which both omit it because their tables have no foreign keys. locations
// does: customer_id references customers(id) ON DELETE RESTRICT, so a
// foreign key violation is a real, reachable outcome here in both
// directions — creating/updating a Location with a CustomerID that does
// not exist, and deleting a Customer that still has Locations — and both
// map to apperror.KindConflict for the same reasoning
// internal/inventory/postgres/errors.go's translateError already gives:
// they are, at heart, the same kind of problem (the request conflicts
// with the current relational state of the data), and distinguishing them
// would require guessing intent for no practical benefit since callers
// already know whether they just called Create or Delete.
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
