// Package service is the Access Network domain's business logic layer.
// It sits between the HTTP layer and the repository layer: HTTP handlers
// never call a repository directly (see internal/accessnetwork/httpapi),
// and repositories never validate or otherwise reason about business
// rules (see internal/accessnetwork/postgres, which trusts its caller) —
// this is where those two responsibilities meet. It mirrors
// internal/catalog/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
)

// AccessNetworkService is the Access Network domain's business logic.
//
// It depends only on accessnetwork.AccessNetworkRepository — not
// clock.Clock, for the same reason internal/catalog/service.CatalogService
// does not: timestamps are already the repository's responsibility, and
// this service has no business rule that needs to reason about "now".
// Goal 6 is explicit that this milestone adds no business rules beyond
// validation, so Get/List/Delete are pure delegation and Create/Update
// are validate-then-delegate — nothing more.
type AccessNetworkService struct {
	networks accessnetwork.AccessNetworkRepository
}

// NewAccessNetworkService builds an AccessNetworkService.
func NewAccessNetworkService(networks accessnetwork.AccessNetworkRepository) *AccessNetworkService {
	return &AccessNetworkService{networks: networks}
}

// Get retrieves an AccessNetwork by ID.
func (s *AccessNetworkService) Get(ctx context.Context, id uuid.UUID) (accessnetwork.AccessNetwork, error) {
	return s.networks.Get(ctx, id)
}

// List returns every AccessNetwork.
func (s *AccessNetworkService) List(ctx context.Context) ([]accessnetwork.AccessNetwork, error) {
	return s.networks.List(ctx)
}

// Create validates a and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of AccessNetworkService —
// so every caller gets the same guarantee for free, and invalid input
// never costs a database round trip. See
// internal/catalog/service.CatalogService.Create for the identical
// reasoning applied to Catalogs.
func (s *AccessNetworkService) Create(ctx context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	if err := a.Validate(); err != nil {
		return accessnetwork.AccessNetwork{}, err
	}
	return s.networks.Create(ctx, a)
}

// Update validates a and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *AccessNetworkService) Update(ctx context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	if err := a.Validate(); err != nil {
		return accessnetwork.AccessNetwork{}, err
	}
	return s.networks.Update(ctx, a)
}

// Delete removes the AccessNetwork identified by id.
func (s *AccessNetworkService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.networks.Delete(ctx, id)
}
