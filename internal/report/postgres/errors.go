package postgres

import "github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"

// translateError wraps a lower-level error into a platform apperror so
// pgx-specific error types never leak past this package. Simpler than
// every other domain's postgres/errors.go (see e.g.
// internal/event/postgres/errors.go): those exist to recognize a unique
// or foreign-key constraint violation on a write and turn it into
// apperror.KindConflict, but this package never writes anything (see
// internal/report's own doc comment), so no such violation is ever
// possible here — every failure is an unexpected one.
func translateError(op string, err error) error {
	if err == nil {
		return nil
	}
	return apperror.Internal(op, err)
}
