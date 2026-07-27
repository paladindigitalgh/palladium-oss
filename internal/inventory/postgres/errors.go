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
// github.com/jackc/pgerrcode) to avoid a dependency for two stable strings;
// add that package instead if more codes need distinguishing later.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// translateError maps a lower-level error into a platform apperror so
// PostgreSQL- and pgx-specific error types never leak past this package.
// op names the operation that failed, for context in the wrapped message.
//
// Callers check for pgx.ErrNoRows themselves before calling this function:
// "no rows" is an expected, meaningful outcome (translated to
// apperror.NotFound with a message naming the record that was missing),
// not a failure this generic translator should guess a message for.
//
// A foreign key violation is also mapped to apperror.KindConflict. It
// arises in two directions — inserting/updating a row whose parent
// reference does not exist, and deleting a row that something else still
// references (see the ON DELETE RESTRICT policy on every inventory foreign
// key, chosen specifically so this is a rejected, recoverable Conflict
// rather than a silent cascade) — and both are, at heart, the same kind of
// problem: the request conflicts with the current relational state of the
// data. Distinguishing them would require guessing intent from the SQL
// text or a second parameter threaded through every call site for no
// practical benefit, since callers already have that context (they know
// whether they just called Create or Delete).
//
// Every other error is classified as apperror.KindInternal and wraps the
// original error via apperror.Wrap. Preserving the cause here (rather than
// discarding it, as NotFound and Conflict do) is deliberate: these are
// unexpected failures, and losing the underlying pgx/Postgres detail would
// make them undebuggable. Callers still only ever see an *apperror.Error —
// they are never forced to type-assert a pgconn.PgError to handle it — so
// this satisfies "don't leak Postgres-specific errors" without sacrificing
// observability for the genuinely unavoidable case.
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
