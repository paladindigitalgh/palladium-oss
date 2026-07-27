// Package postgres implements the auth domain's UserRepository against
// PostgreSQL using pgx directly — no ORM — following the exact pattern
// established by internal/inventory/postgres.SiteRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// UserRepository implements auth.UserRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated here.
type UserRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ auth.UserRepository = (*UserRepository)(nil)

// NewUserRepository builds a UserRepository.
func NewUserRepository(db database.Querier, clock clock.Clock, ids id.Generator) *UserRepository {
	return &UserRepository{db: db, clock: clock, ids: ids}
}

// GetByID retrieves a User by ID, or an apperror.KindNotFound error if
// none exists.
func (r *UserRepository) GetByID(ctx context.Context, userID uuid.UUID) (auth.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user, err := scanUser(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, userNotFoundByID(userID)
		}
		return auth.User{}, translateError("get user", err)
	}
	return user, nil
}

// GetByEmail retrieves a User by email, or an apperror.KindNotFound error
// if none exists. This is the lookup a login attempt starts from.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (auth.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user, err := scanUser(r.db.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, apperror.NotFound(fmt.Sprintf("user with email %s not found", email))
		}
		return auth.User{}, translateError("get user by email", err)
	}
	return user, nil
}

// Create inserts user and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt, and
// UpdatedAt itself; any values already set on the input User for those
// fields are ignored. An Email that collides with an existing User fails
// with an apperror.KindConflict error (see translateError). Create does
// not hash PasswordHash — it stores exactly the string it is given, which
// must already be a bcrypt hash produced by auth.HashPassword — nor does
// it decide Role: both are taken from the input User exactly as given.
// The repository has no business logic and does not know how a password
// became a hash or why a caller chose a particular Role; deciding that is
// e.g. internal/auth/bootstrap's job (it always sets RoleAdministrator),
// not this one's.
func (r *UserRepository) Create(ctx context.Context, user auth.User) (auth.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, email, password_hash, role, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanUser(r.db.QueryRow(ctx, query,
		r.ids.New(), user.Email, user.PasswordHash, string(user.Role), now))
	if err != nil {
		return auth.User{}, translateError("create user", err)
	}
	return created, nil
}

// UpdatePasswordHash overwrites the PasswordHash of the User identified by
// id and returns the persisted record, or an apperror.KindNotFound error
// if it does not exist.
//
// CreatedAt cannot be altered through this method, for the same reason as
// SiteRepository.Update: the UPDATE statement below never assigns that
// column. Email is also never touched here — UserRepository has no method
// that changes it, since nothing in this milestone needs to (see
// repository.go).
//
// The parameter is named userID, not id, even though the interface in
// repository.go names it id: an implementation is free to choose its own
// parameter names, and site.go/building.go/etc. all avoid id specifically
// because it would shadow the imported internal/platform/id package.
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) (auth.User, error) {
	const query = `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, email, password_hash, role, created_at, updated_at
	`

	updated, err := scanUser(r.db.QueryRow(ctx, query, passwordHash, r.clock.Now(), userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, userNotFoundByID(userID)
		}
		return auth.User{}, translateError("update user password hash", err)
	}
	return updated, nil
}

// Count returns how many Users exist. See the UserRepository interface's
// doc comment in internal/auth/repository.go for why this exists despite
// there being no List: it backs internal/auth/bootstrap's "refuse if a
// user already exists" check, and nothing else needs it yet.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	const query = `SELECT count(*) FROM users`

	var count int
	if err := r.db.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, translateError("count users", err)
	}
	return count, nil
}

func userNotFoundByID(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("user %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan). This
// package only ever fetches one row at a time (there is no List), but the
// interface is kept for consistency with
// internal/inventory/postgres — and in case a future List is added here.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (auth.User, error) {
	var (
		user auth.User
		role string
	)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &role, &user.CreatedAt, &user.UpdatedAt)
	user.Role = auth.Role(role)
	return user, err
}
