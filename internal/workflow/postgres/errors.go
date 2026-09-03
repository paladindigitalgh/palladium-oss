package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// translateError maps a lower-level error into a platform apperror,
// mirroring every other domain's postgres/errors.go in this codebase.
// workflow_instances.service_id and .requested_by_user_id both reference
// their parent tables ON DELETE RESTRICT.
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
