// Package service is the PON Port domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/ponport/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/ponport/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors internal/olt/service
// exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// PONPortService is the PON Port domain's business logic.
//
// It depends only on ponport.PONPortRepository — not clock.Clock, for
// the same reason internal/olt/service.OLTService does not: timestamps
// are already the repository's responsibility, and this service has no
// business rule that needs to reason about "now". Goal 6 is explicit
// that this milestone adds no business rules beyond validation, so
// Get/List/Delete are pure delegation and Create/Update are
// validate-then-delegate — nothing more.
type PONPortService struct {
	ports ponport.PONPortRepository
}

// NewPONPortService builds a PONPortService.
func NewPONPortService(ports ponport.PONPortRepository) *PONPortService {
	return &PONPortService{ports: ports}
}

// Get retrieves a PONPort by ID.
func (s *PONPortService) Get(ctx context.Context, id uuid.UUID) (ponport.PONPort, error) {
	return s.ports.Get(ctx, id)
}

// List returns every PONPort.
func (s *PONPortService) List(ctx context.Context) ([]ponport.PONPort, error) {
	return s.ports.List(ctx)
}

// Create validates p and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of PONPortService — so
// every caller gets the same guarantee for free, and invalid input never
// costs a database round trip. See
// internal/olt/service.OLTService.Create for the identical reasoning
// applied to OLTs.
func (s *PONPortService) Create(ctx context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	if err := p.Validate(); err != nil {
		return ponport.PONPort{}, err
	}
	return s.ports.Create(ctx, p)
}

// Update validates p and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *PONPortService) Update(ctx context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	if err := p.Validate(); err != nil {
		return ponport.PONPort{}, err
	}
	return s.ports.Update(ctx, p)
}

// Delete removes the PONPort identified by id.
func (s *PONPortService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.ports.Delete(ctx, id)
}
