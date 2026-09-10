package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// Postgres error codes this package distinguishes. See:
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgForeignKeyViolation = "23503"
)

// translateError maps a lower-level error into a platform apperror so
// PostgreSQL- and pgx-specific error types never leak past this package.
// op names the operation that failed, for context in the wrapped message.
//
// notes.author_user_id references users(id) ON DELETE RESTRICT (see the
// migration), so creating a Note with an AuthorUserID that does not
// exist is a real, reachable outcome, and it maps to
// apperror.KindConflict — the same reasoning
// internal/contact/postgres/errors.go's translateError gives for its own
// foreign-key branch. There is no unique-violation branch: unlike
// Contact, nothing about a Note is meant to be unique.
func translateError(op string, err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgForeignKeyViolation {
			return apperror.Conflict(fmt.Sprintf("%s: violates a foreign key relationship", op))
		}
	}

	return apperror.Internal(op, err)
}
