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
		SELECT id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
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
		SELECT id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
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

// List returns every User, ordered by email for stable, human-useful
// output — the same reasoning as internal/provider/postgres.List backing
// User Management's browse view.
func (r *UserRepository) List(ctx context.Context) ([]auth.User, error) {
	const query = `
		SELECT id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
		FROM users
		ORDER BY email
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list users", err)
	}
	defer rows.Close()

	users := []auth.User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, translateError("scan user row", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list users", err)
	}

	return users, nil
}

// Create inserts user and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt, and
// UpdatedAt itself; any values already set on the input User for those
// fields are ignored. An Email that collides with an existing User fails
// with an apperror.KindConflict error (see translateError). Create does
// not hash PasswordHash — it stores exactly the string it is given, which
// must already be a bcrypt hash produced by auth.HashPassword — nor does
// it decide Role or Status: all three are taken from the input User
// exactly as given, as are the optional FirstName/LastName. The
// repository has no business logic and does not know how a password
// became a hash or why a caller chose a particular Role or Status;
// deciding that is e.g. internal/auth/bootstrap's job (it always sets
// RoleAdministrator/UserStatusActive) or
// internal/auth/service.UserManagementService's (it always sets
// UserStatusActive), not this one's.
func (r *UserRepository) Create(ctx context.Context, user auth.User) (auth.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanUser(r.db.QueryRow(ctx, query,
		r.ids.New(), user.Email, user.PasswordHash, user.FirstName, user.LastName,
		string(user.Role), string(user.Status), now))
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
// repository.go); FirstName/LastName are similarly untouched — that is
// UpdateName's job below, not this one's.
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
		RETURNING id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
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

// UpdateName overwrites the FirstName/LastName of the User identified by
// userID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist. Both are optional (see model.go's doc
// comment on User) — an empty string clears a name that was previously
// set, the same "empty string means unset" semantics Create already
// gives them.
func (r *UserRepository) UpdateName(ctx context.Context, userID uuid.UUID, firstName, lastName string) (auth.User, error) {
	const query = `
		UPDATE users
		SET first_name = $1, last_name = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
	`

	updated, err := scanUser(r.db.QueryRow(ctx, query, firstName, lastName, r.clock.Now(), userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, userNotFoundByID(userID)
		}
		return auth.User{}, translateError("update user name", err)
	}
	return updated, nil
}

// UpdateRole overwrites the Role of the User identified by userID and
// returns the persisted record, or an apperror.KindNotFound error if it
// does not exist. Whether this change is safe to make (e.g. it would not
// remove the last active Administrator) is
// internal/auth/service.UserManagementService's job, not this
// repository's — it trusts its caller, same as every other repository in
// this codebase.
func (r *UserRepository) UpdateRole(ctx context.Context, userID uuid.UUID, role auth.Role) (auth.User, error) {
	const query = `
		UPDATE users
		SET role = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
	`

	updated, err := scanUser(r.db.QueryRow(ctx, query, string(role), r.clock.Now(), userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, userNotFoundByID(userID)
		}
		return auth.User{}, translateError("update user role", err)
	}
	return updated, nil
}

// UpdateStatus overwrites the Status of the User identified by userID and
// returns the persisted record, or an apperror.KindNotFound error if it
// does not exist. See UpdateRole's doc comment for why this repository
// does not itself guard against locking out the last Administrator.
func (r *UserRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, status auth.UserStatus) (auth.User, error) {
	const query = `
		UPDATE users
		SET status = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, email, password_hash, first_name, last_name, role, status, created_at, updated_at
	`

	updated, err := scanUser(r.db.QueryRow(ctx, query, string(status), r.clock.Now(), userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, userNotFoundByID(userID)
		}
		return auth.User{}, translateError("update user status", err)
	}
	return updated, nil
}

// Count returns how many Users exist. See the UserRepository interface's
// doc comment in internal/auth/repository.go for why this exists
// alongside List: it backs internal/auth/bootstrap's "refuse if a user
// already exists" check specifically, and predates List.
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
		user   auth.User
		role   string
		status string
	)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&role, &status, &user.CreatedAt, &user.UpdatedAt)
	user.Role = auth.Role(role)
	user.Status = auth.UserStatus(status)
	return user, err
}
