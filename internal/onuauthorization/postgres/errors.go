package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// Postgres error codes this package distinguishes. See
// internal/accessattachment/postgres/errors.go for why these are
// hardcoded rather than pulled from a constants package.
const (
	pgForeignKeyViolation = "23503"
)

// translateError maps a lower-level error into a platform apperror so
// PostgreSQL- and pgx-specific error types never leak past this package.
// op names the operation that failed, for context in the wrapped
// message. Mirrors internal/accessattachment/postgres/errors.go's own
// translateError, minus the unique-violation branch: onu_authorizations
// has no unique constraint (see this migration's own comment on why
// "at most one active authorization per device" is application logic,
// not a database constraint, mirroring access_attachments' identical
// choice).
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
