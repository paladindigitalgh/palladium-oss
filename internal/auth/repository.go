package auth

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository persists Users. It follows the same shape as the
// Inventory repositories (internal/inventory/repository.go) — Create
// returns the persisted entity, nothing here implements it directly, no
// storage technology is implied — with two differences that reflect what
// this milestone actually needs:
//
//   - GetByEmail exists because a login attempt starts from an email, not
//     an ID, and this is a direct lookup rather than List-and-filter.
//   - There is no List and no Delete: this milestone establishes login
//     only, not user administration. UpdatePasswordHash is the one
//     mutation login-adjacent flows need (e.g. a future "change password"
//     use case); nothing yet needs to change Email or to remove a User.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	Create(ctx context.Context, user User) (User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) (User, error)
}
