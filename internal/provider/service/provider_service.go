// Package service is the Provider domain's business logic layer. It
// sits between the HTTP layer and the repository layer: HTTP handlers
// never call a repository directly (see internal/provider/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/provider/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/serviceprofile/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provider"
)

// ProviderService is the Provider domain's business logic.
type ProviderService struct {
	providers provider.ProviderRepository
}

// NewProviderService builds a ProviderService.
func NewProviderService(providers provider.ProviderRepository) *ProviderService {
	return &ProviderService{providers: providers}
}

// Get retrieves a Provider by ID.
func (s *ProviderService) Get(ctx context.Context, id uuid.UUID) (provider.Provider, error) {
	return s.providers.Get(ctx, id)
}

// List returns every Provider.
func (s *ProviderService) List(ctx context.Context) ([]provider.Provider, error) {
	return s.providers.List(ctx)
}

// Create validates p and, if valid, persists it. See
// serviceprofile/service.ServiceProfileService.Create for why validation
// happens here rather than in the handler or repository.
func (s *ProviderService) Create(ctx context.Context, p provider.Provider) (provider.Provider, error) {
	if err := p.Validate(); err != nil {
		return provider.Provider{}, err
	}
	return s.providers.Create(ctx, p)
}

// Update validates p and, if valid, persists the change.
func (s *ProviderService) Update(ctx context.Context, p provider.Provider) (provider.Provider, error) {
	if err := p.Validate(); err != nil {
		return provider.Provider{}, err
	}
	return s.providers.Update(ctx, p)
}

// Delete removes the Provider identified by id.
func (s *ProviderService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.providers.Delete(ctx, id)
}
