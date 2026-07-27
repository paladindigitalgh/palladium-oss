package auth

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository persists Users. It follows the same shape as the
// Inventory repositories (internal/inventory/repository.go) — Create
// returns the persisted entity, nothing here implements it directly, no
// storage technology is implied — with differences that reflect what the
// auth domain actually needs so far:
//
//   - GetByEmail exists because a login attempt starts from an email, not
//     an ID, and this is a direct lookup rather than List-and-filter.
//   - There is no List and no Delete: this is login and installation
//     bootstrapping, not user administration. UpdatePasswordHash is the
//     one mutation login-adjacent flows need (e.g. a future "change
//     password" use case); nothing yet needs to change Email or to
//     remove a User.
//   - Count exists solely for internal/auth/bootstrap's "refuse to create
//     an administrator if a user already exists" check. It is not the
//     first step toward a List/browse-users feature: it returns a number,
//     never User data, and answers exactly one question ("does any user
//     exist yet?") rather than supporting pagination or search.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	Create(ctx context.Context, user User) (User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) (User, error)
	Count(ctx context.Context) (int, error)
}
