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
// The unique-violation branch is reachable directly against this
// package's own schema: authentication_methods.name is UNIQUE (this
// milestone's explicit "Name unique" rule — see
// database/migrations/00023_authentication_authentication_methods.sql).
// The foreign-key-violation branch is written in from the start even
// though nothing in the authentication_methods table itself references
// another table: connection_profiles.authentication_id will reference
// authentication_methods(id) ON DELETE RESTRICT (see
// internal/connectionprofile/postgres and
// database/migrations/00024_connectionprofile_connection_profiles.sql),
// so deleting an Authentication that a ConnectionProfile still
// references is a real, reachable outcome against this schema. This is
// the same lesson applied proactively throughout this codebase since
// internal/customer/postgres/errors.go originally omitted it and had to
// be corrected after the fact.
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
