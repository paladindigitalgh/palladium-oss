// Package service is the Access Interface domain's business logic
// layer. It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/accessinterface/httpapi), and repositories never validate or
// otherwise reason about business rules (see
// internal/accessinterface/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors internal/olt/service
// exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
)

// AccessInterfaceService is the Access Interface domain's business
// logic.
//
// It depends only on accessinterface.AccessInterfaceRepository — not
// clock.Clock, for the same reason internal/olt/service.OLTService does
// not: timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now". This
// milestone adds no business rules to AccessInterface beyond validation
// (the active-attachment uniqueness rule belongs to
// internal/accessattachment/service, one layer up), so Get/List/Delete
// are pure delegation and Create/Update are validate-then-delegate —
// nothing more.
type AccessInterfaceService struct {
	interfaces accessinterface.AccessInterfaceRepository
}

// NewAccessInterfaceService builds an AccessInterfaceService.
func NewAccessInterfaceService(interfaces accessinterface.AccessInterfaceRepository) *AccessInterfaceService {
	return &AccessInterfaceService{interfaces: interfaces}
}

// Get retrieves an AccessInterface by ID.
func (s *AccessInterfaceService) Get(ctx context.Context, id uuid.UUID) (accessinterface.AccessInterface, error) {
	return s.interfaces.Get(ctx, id)
}

// List returns every AccessInterface.
func (s *AccessInterfaceService) List(ctx context.Context) ([]accessinterface.AccessInterface, error) {
	return s.interfaces.List(ctx)
}

// Create validates a and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of
// AccessInterfaceService — so every caller gets the same guarantee for
// free, and invalid input never costs a database round trip. See
// internal/olt/service.OLTService.Create for the identical reasoning
// applied to OLTs.
func (s *AccessInterfaceService) Create(ctx context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	if err := a.Validate(); err != nil {
		return accessinterface.AccessInterface{}, err
	}
	return s.interfaces.Create(ctx, a)
}

// Update validates a and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *AccessInterfaceService) Update(ctx context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	if err := a.Validate(); err != nil {
		return accessinterface.AccessInterface{}, err
	}
	return s.interfaces.Update(ctx, a)
}

// Delete removes the AccessInterface identified by id.
func (s *AccessInterfaceService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.interfaces.Delete(ctx, id)
}
