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
//   - List backs User Management (internal/auth/service.UserManagementService)
//     browsing every account — the milestone that added it explicitly
//     needed a browse-users feature, unlike when this package was
//     login-and-bootstrap only.
//   - There is no Delete: Users are never removed, only deactivated (see
//     UpdateStatus) — both events.actor_user_id and
//     workflow_instances.requested_by_user_id reference users(id) with
//     ON DELETE RESTRICT, so a real delete would fail for any User who
//     has ever done anything. UpdatePasswordHash, UpdateName, UpdateRole,
//     and UpdateStatus are the mutations User Management, Profile, and
//     login-adjacent flows need; nothing yet needs to change Email.
//   - Count exists solely for internal/auth/bootstrap's "refuse to create
//     an administrator if a user already exists" check. It predates List
//     and was never generalized into it: Count answers exactly one
//     question ("does any user exist yet?") without needing User data at
//     all.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context) ([]User, error)
	Create(ctx context.Context, user User) (User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) (User, error)
	UpdateName(ctx context.Context, id uuid.UUID, firstName, lastName string) (User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role Role) (User, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status UserStatus) (User, error)
	Count(ctx context.Context) (int, error)
}
