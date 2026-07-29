// Package service is the Authentication domain's business logic layer.
// It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/authentication/httpapi), and repositories never validate or
// otherwise reason about business rules (see
// internal/authentication/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/catalog/service exactly.
//
// This service never touches encryption, or Password/PrivateKey's
// storage representation at all — see
// internal/authentication/model.go's doc comment, "Plaintext in memory,
// ciphertext at rest": encryption is a repository-layer concern, this
// layer only validates plaintext and delegates.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
)

// AuthenticationService is the Authentication domain's business logic.
//
// It depends only on authentication.AuthenticationRepository — not
// clock.Clock, for the same reason internal/catalog/service.CatalogService
// does not: timestamps are already the repository's responsibility, and
// this service has no business rule that needs to reason about "now". It
// also does not depend on internal/platform/encryption directly, for the
// reason this package's own doc comment gives.
type AuthenticationService struct {
	authentications authentication.AuthenticationRepository
}

// NewAuthenticationService builds an AuthenticationService.
func NewAuthenticationService(authentications authentication.AuthenticationRepository) *AuthenticationService {
	return &AuthenticationService{authentications: authentications}
}

// Get retrieves an Authentication by ID.
func (s *AuthenticationService) Get(ctx context.Context, id uuid.UUID) (authentication.Authentication, error) {
	return s.authentications.Get(ctx, id)
}

// List returns every Authentication.
func (s *AuthenticationService) List(ctx context.Context) ([]authentication.Authentication, error) {
	return s.authentications.List(ctx)
}

// Create validates a and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of AuthenticationService
// — so every caller gets the same guarantee for free, and invalid input
// never costs a database round trip. See
// internal/catalog/service.CatalogService.Create for the identical
// reasoning applied to Catalogs.
func (s *AuthenticationService) Create(ctx context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	if err := a.Validate(); err != nil {
		return authentication.Authentication{}, err
	}
	return s.authentications.Create(ctx, a)
}

// Update validates a and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *AuthenticationService) Update(ctx context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	if err := a.Validate(); err != nil {
		return authentication.Authentication{}, err
	}
	return s.authentications.Update(ctx, a)
}

// Delete removes the Authentication identified by id.
func (s *AuthenticationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.authentications.Delete(ctx, id)
}
