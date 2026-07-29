// Package service is the Connection Profile domain's business logic
// layer. It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/connectionprofile/httpapi), and repositories never validate
// or otherwise reason about business rules (see
// internal/connectionprofile/postgres, which trusts its caller) — this
// is where those two responsibilities meet. It mirrors
// internal/catalog/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
)

// ConnectionProfileService is the Connection Profile domain's business
// logic.
//
// It depends only on connectionprofile.ConnectionProfileRepository —
// not clock.Clock, for the same reason
// internal/catalog/service.CatalogService does not: timestamps are
// already the repository's responsibility, and this service has no
// business rule that needs to reason about "now".
type ConnectionProfileService struct {
	profiles connectionprofile.ConnectionProfileRepository
}

// NewConnectionProfileService builds a ConnectionProfileService.
func NewConnectionProfileService(profiles connectionprofile.ConnectionProfileRepository) *ConnectionProfileService {
	return &ConnectionProfileService{profiles: profiles}
}

// Get retrieves a ConnectionProfile by ID.
func (s *ConnectionProfileService) Get(ctx context.Context, id uuid.UUID) (connectionprofile.ConnectionProfile, error) {
	return s.profiles.Get(ctx, id)
}

// List returns every ConnectionProfile.
func (s *ConnectionProfileService) List(ctx context.Context) ([]connectionprofile.ConnectionProfile, error) {
	return s.profiles.List(ctx)
}

// Create validates p and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of
// ConnectionProfileService — so every caller gets the same guarantee for
// free, and invalid input never costs a database round trip. See
// internal/catalog/service.CatalogService.Create for the identical
// reasoning applied to Catalogs.
func (s *ConnectionProfileService) Create(ctx context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	if err := p.Validate(); err != nil {
		return connectionprofile.ConnectionProfile{}, err
	}
	return s.profiles.Create(ctx, p)
}

// Update validates p and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *ConnectionProfileService) Update(ctx context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	if err := p.Validate(); err != nil {
		return connectionprofile.ConnectionProfile{}, err
	}
	return s.profiles.Update(ctx, p)
}

// Delete removes the ConnectionProfile identified by id.
func (s *ConnectionProfileService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.profiles.Delete(ctx, id)
}
