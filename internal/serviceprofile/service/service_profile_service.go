// Package service is the Service Profile domain's business logic layer.
// It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/serviceprofile/httpapi), and repositories never validate or
// otherwise reason about business rules (see
// internal/serviceprofile/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/catalog/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
)

// ServiceProfileService is the Service Profile domain's business logic.
//
// It depends only on serviceprofile.ServiceProfileRepository — not
// clock.Clock, for the same reason internal/catalog/service.CatalogService
// does not: timestamps are already the repository's responsibility, and
// this service has no business rule that needs to reason about "now".
type ServiceProfileService struct {
	profiles serviceprofile.ServiceProfileRepository
}

// NewServiceProfileService builds a ServiceProfileService.
func NewServiceProfileService(profiles serviceprofile.ServiceProfileRepository) *ServiceProfileService {
	return &ServiceProfileService{profiles: profiles}
}

// Get retrieves a ServiceProfile by ID.
func (s *ServiceProfileService) Get(ctx context.Context, id uuid.UUID) (serviceprofile.ServiceProfile, error) {
	return s.profiles.Get(ctx, id)
}

// List returns every ServiceProfile.
func (s *ServiceProfileService) List(ctx context.Context) ([]serviceprofile.ServiceProfile, error) {
	return s.profiles.List(ctx)
}

// Create validates p and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of ServiceProfileService
// — so every caller gets the same guarantee for free, and invalid input
// never costs a database round trip. See
// internal/catalog/service.CatalogService.Create for the identical
// reasoning applied to Catalogs.
func (s *ServiceProfileService) Create(ctx context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	if err := p.Validate(); err != nil {
		return serviceprofile.ServiceProfile{}, err
	}
	return s.profiles.Create(ctx, p)
}

// Update validates p and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *ServiceProfileService) Update(ctx context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	if err := p.Validate(); err != nil {
		return serviceprofile.ServiceProfile{}, err
	}
	return s.profiles.Update(ctx, p)
}

// Delete removes the ServiceProfile identified by id.
func (s *ServiceProfileService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.profiles.Delete(ctx, id)
}
