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
// Hardcoded rather than pulled from a constants package, mirroring
// internal/serviceequipment/postgres/errors.go's own reasoning.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// translateError maps a lower-level error into a platform apperror so
// PostgreSQL- and pgx-specific error types never leak past this package.
// op names the operation that failed, for context in the wrapped
// message. Mirrors internal/serviceequipment/postgres/errors.go's own
// translateError exactly: customer_devices.customer_id and
// customer_devices.device_id both reference their parent tables ON
// DELETE RESTRICT, so a foreign key violation is a real, reachable
// outcome in the same three directions that file documents.
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
