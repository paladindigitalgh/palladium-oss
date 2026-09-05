// Package service is the ProvisioningProfile domain's business logic
// layer. It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/provisioning/httpapi), and repositories never validate or
// otherwise reason about business rules (see internal/provisioning/postgres,
// which trusts its caller) — this is where those two responsibilities
// meet. It mirrors internal/product/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// ProvisioningProfileService is the ProvisioningProfile domain's business
// logic.
type ProvisioningProfileService struct {
	profiles provisioning.ProvisioningProfileRepository
}

// NewProvisioningProfileService builds a ProvisioningProfileService.
func NewProvisioningProfileService(profiles provisioning.ProvisioningProfileRepository) *ProvisioningProfileService {
	return &ProvisioningProfileService{profiles: profiles}
}

// Get retrieves a ProvisioningProfile by ID.
func (s *ProvisioningProfileService) Get(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningProfile, error) {
	return s.profiles.Get(ctx, id)
}

// List returns every ProvisioningProfile.
func (s *ProvisioningProfileService) List(ctx context.Context) ([]provisioning.ProvisioningProfile, error) {
	return s.profiles.List(ctx)
}

// Create validates p and, if valid, persists it. See
// product/service.ProductService.Create for why validation happens here
// rather than in the handler or repository.
func (s *ProvisioningProfileService) Create(ctx context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	if err := p.Validate(); err != nil {
		return provisioning.ProvisioningProfile{}, err
	}
	return s.profiles.Create(ctx, p)
}

// Update validates p and, if valid, persists the change.
func (s *ProvisioningProfileService) Update(ctx context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	if err := p.Validate(); err != nil {
		return provisioning.ProvisioningProfile{}, err
	}
	return s.profiles.Update(ctx, p)
}

// Delete removes the ProvisioningProfile identified by id.
func (s *ProvisioningProfileService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.profiles.Delete(ctx, id)
}
