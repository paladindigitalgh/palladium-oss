package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// ProfileService is a signed-in User's self-service business logic:
// viewing their own record, changing their own display name, and
// changing their own password. It is a separate type from
// UserManagementService, not another method on it, because it answers a
// different question — "what can I do to my own account" rather than
// "what can an Administrator do to any account" — and the two have no
// overlapping guard logic: UpdateName and ChangePassword never need
// guardLastActiveAdministrator, since neither can change a Role or a
// Status.
type ProfileService struct {
	users auth.UserRepository
}

// NewProfileService builds a ProfileService.
func NewProfileService(users auth.UserRepository) *ProfileService {
	return &ProfileService{users: users}
}

// Get returns the User identified by id.
func (s *ProfileService) Get(ctx context.Context, id uuid.UUID) (auth.User, error) {
	return s.users.GetByID(ctx, id)
}

// UpdateName changes the FirstName/LastName of the User identified by id.
// Both are optional (see auth.User's doc comment) — an empty string
// clears a name that was previously set.
func (s *ProfileService) UpdateName(ctx context.Context, id uuid.UUID, firstName, lastName string) (auth.User, error) {
	return s.users.UpdateName(ctx, id, firstName, lastName)
}

// ChangePassword verifies currentPassword against the User identified by
// id's stored hash and, if it matches, replaces it with a hash of
// newPassword.
//
// Requiring currentPassword (rather than trusting the caller's JWT alone)
// guards against a stolen, still-valid session token being enough to lock
// the real account holder out by itself — the same "prove you still know
// the secret" reasoning a password change form protects on any system.
func (s *ProfileService) ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) (auth.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return auth.User{}, err
	}

	if !auth.VerifyPassword(user.PasswordHash, currentPassword) {
		return auth.User{}, apperror.Invalid("current password is incorrect")
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return auth.User{}, err
	}

	return s.users.UpdatePasswordHash(ctx, id, hash)
}
