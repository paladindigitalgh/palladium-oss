// Package service is User Management's business logic layer: creating
// accounts, listing them, changing a Role, and deactivating/reactivating
// an account. It sits between the HTTP layer and internal/auth's
// UserRepository, the same separation internal/provider/service draws
// for Provider — HTTP handlers never call a repository directly (see
// internal/auth/httpapi), and the repository never reasons about
// business rules (see internal/auth/postgres, which trusts its caller).
//
// This is a separate package from internal/auth itself, unlike every
// other domain's service package sitting directly alongside its
// repository interface: internal/auth already holds AuthService, tightly
// bound to login internals (see auth/service.go's own doc comment on why
// it has no Register/CreateUser method). A second service for admin CRUD
// belongs in its own subpackage rather than growing AuthService into a
// mixed bag of login and administration concerns.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// UserManagementService is User Management's business logic.
type UserManagementService struct {
	users auth.UserRepository
}

// NewUserManagementService builds a UserManagementService.
func NewUserManagementService(users auth.UserRepository) *UserManagementService {
	return &UserManagementService{users: users}
}

// List returns every User.
func (s *UserManagementService) List(ctx context.Context) ([]auth.User, error) {
	return s.users.List(ctx)
}

// Create hashes password and persists a new User with the given email,
// role, and optional first/last name.
//
// Status is never a caller choice — a freshly created account always
// starts UserStatusActive, the same reasoning
// internal/provider/httpapi/dto.go documents for why creating a Provider
// always stamps Status "Active": this is a form that creates a fresh,
// currently-usable account, not one that also needs to create a
// pre-deactivated one.
func (s *UserManagementService) Create(ctx context.Context, email, password, firstName, lastName string, role auth.Role) (auth.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return auth.User{}, err
	}

	user := auth.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
		Role:         role,
		Status:       auth.UserStatusActive,
	}
	if err := user.Validate(); err != nil {
		return auth.User{}, err
	}

	return s.users.Create(ctx, user)
}

// UpdateRole changes the Role of the User identified by id, refusing if
// doing so would leave zero active Administrators (see
// wouldRemoveLastActiveAdministrator).
func (s *UserManagementService) UpdateRole(ctx context.Context, id uuid.UUID, role auth.Role) (auth.User, error) {
	if !role.Valid() {
		return auth.User{}, apperror.Invalid("role is not valid")
	}

	target, err := s.users.GetByID(ctx, id)
	if err != nil {
		return auth.User{}, err
	}

	wouldRemainAdministrator := role == auth.RoleAdministrator
	if err := s.guardLastActiveAdministrator(ctx, target, wouldRemainAdministrator); err != nil {
		return auth.User{}, err
	}

	return s.users.UpdateRole(ctx, id, role)
}

// Deactivate sets the User identified by id to UserStatusInactive,
// refusing if doing so would leave zero active Administrators (see
// wouldRemoveLastActiveAdministrator). A deactivated User cannot log in
// (see AuthService.Authenticate) and loses access to every
// capability-gated endpoint on their very next request (see
// authz.Middleware.Require), without deleting any record referencing
// them.
func (s *UserManagementService) Deactivate(ctx context.Context, id uuid.UUID) (auth.User, error) {
	target, err := s.users.GetByID(ctx, id)
	if err != nil {
		return auth.User{}, err
	}

	if err := s.guardLastActiveAdministrator(ctx, target, false); err != nil {
		return auth.User{}, err
	}

	return s.users.UpdateStatus(ctx, id, auth.UserStatusInactive)
}

// Reactivate sets the User identified by id back to UserStatusActive.
// Unlike Deactivate and UpdateRole, this never needs
// guardLastActiveAdministrator: reactivating a User can only increase the
// number of active Administrators, never reduce it.
func (s *UserManagementService) Reactivate(ctx context.Context, id uuid.UUID) (auth.User, error) {
	return s.users.UpdateStatus(ctx, id, auth.UserStatusActive)
}

// guardLastActiveAdministrator returns an apperror.KindConflict error if
// target is currently an active Administrator and wouldRemainAdministrator
// is false, and target is the only active Administrator that exists.
//
// This single check covers every dangerous case — self-demotion,
// self-deactivation, demoting or deactivating someone else — without
// needing to know which User is making the request: it only asks "does
// at least one active Administrator remain after this change," which is
// the actual invariant worth protecting, not "is the caller acting on
// themselves."
func (s *UserManagementService) guardLastActiveAdministrator(ctx context.Context, target auth.User, wouldRemainAdministrator bool) error {
	isCurrentlyActiveAdministrator := target.Role == auth.RoleAdministrator && target.Status == auth.UserStatusActive
	if !isCurrentlyActiveAdministrator || wouldRemainAdministrator {
		return nil
	}

	all, err := s.users.List(ctx)
	if err != nil {
		return err
	}

	remaining := 0
	for _, u := range all {
		if u.ID == target.ID {
			continue
		}
		if u.Role == auth.RoleAdministrator && u.Status == auth.UserStatusActive {
			remaining++
		}
	}

	if remaining == 0 {
		return apperror.Conflict("cannot remove the last active administrator")
	}
	return nil
}
