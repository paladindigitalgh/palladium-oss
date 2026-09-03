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
// Hardcoded rather than pulled from a constants package, the same
// reasoning as internal/location/postgres/errors.go.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// translateError maps a lower-level error into a platform apperror so
// PostgreSQL- and pgx-specific error types never leak past this package.
// op names the operation that failed, for context in the wrapped message.
//
// This mirrors internal/location/postgres/errors.go's translateError,
// including the foreign-key-violation branch: contacts.customer_id
// references customers(id), so creating or updating a Contact with a
// CustomerID that does not exist is a real, reachable outcome, and it
// maps to apperror.KindConflict for the same reasoning. Unlike Location,
// the FK is ON DELETE CASCADE, not RESTRICT (see the migration), so
// deleting a Customer never fails on account of a Contact — there is no
// "delete blocked by existing Contact" case to translate here, only the
// create/update-time "CustomerID does not exist" case.
//
// Callers check for pgx.ErrNoRows themselves before calling this
// function, the same reasoning as every other repository in this
// codebase.
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
