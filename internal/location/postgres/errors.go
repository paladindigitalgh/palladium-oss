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

// fkViolationReasons maps the name of a foreign key constraint that can
// fail against this table to a human-readable reason, so a caller sees
// specifically what is still attached instead of a generic "violates a
// foreign key relationship" — which, on a Delete, previously left every
// caller guessing (see e.g. CustomerDetailView.vue's confirmDeleteLocation
// on the frontend, which used to hardcode "still has services attached"
// even when the real blocker was a Device's placement history, not a
// Service at all).
//
// locations_customer_id_fkey is this table's own outgoing reference
// (Create/Update with a CustomerID that does not exist); the other two
// are incoming references from tables whose rows are never deleted, only
// marked inactive/detached (see internal/customer/removal's package doc
// comment and internal/customerdevice's own "un-tie, not delete"
// reasoning) — which is exactly why a Location can still be blocked here
// even after every visible Service and Device attachment looks gone: a
// detached customer_devices row is history, kept forever, and still
// satisfies this constraint.
var fkViolationReasons = map[string]string{
	"locations_customer_id_fkey":        "the customer does not exist",
	"services_location_id_fkey":         "it still has a Service",
	"customer_devices_location_id_fkey": "it still has Device placement history",
}

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
// not exist, and deleting a Location that still has Services or Device
// placement history — and all map to apperror.KindConflict for the same
// reasoning internal/inventory/postgres/errors.go's translateError
// already gives: they are, at heart, the same kind of problem (the
// request conflicts with the current relational state of the data).
// Unlike that package, though, this one does distinguish which
// relationship actually conflicted (see fkViolationReasons) — locations
// has more than one incoming reference, so "a foreign key relationship"
// alone is not specific enough for a caller to act on.
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
			if reason, ok := fkViolationReasons[pgErr.ConstraintName]; ok {
				return apperror.Conflict(fmt.Sprintf("%s: %s", op, reason))
			}
			return apperror.Conflict(fmt.Sprintf("%s: violates a foreign key relationship", op))
		}
	}

	return apperror.Internal(op, err)
}
