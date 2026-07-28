// Package service is the OLT domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/olt/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/olt/postgres, which trusts its caller) — this is where
// those two responsibilities meet. It mirrors internal/product/service
// exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
)

// OLTService is the OLT domain's business logic.
//
// It depends only on olt.OLTRepository — not clock.Clock, for the same
// reason internal/product/service.ProductService does not: timestamps
// are already the repository's responsibility, and this service has no
// business rule that needs to reason about "now". Goal 6 is explicit
// that this milestone adds no business rules beyond validation, so
// Get/List/Delete are pure delegation and Create/Update are
// validate-then-delegate — nothing more.
type OLTService struct {
	olts olt.OLTRepository
}

// NewOLTService builds an OLTService.
func NewOLTService(olts olt.OLTRepository) *OLTService {
	return &OLTService{olts: olts}
}

// Get retrieves an OLT by ID.
func (s *OLTService) Get(ctx context.Context, id uuid.UUID) (olt.OLT, error) {
	return s.olts.Get(ctx, id)
}

// List returns every OLT.
func (s *OLTService) List(ctx context.Context) ([]olt.OLT, error) {
	return s.olts.List(ctx)
}

// Create validates o and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of OLTService — so every
// caller gets the same guarantee for free, and invalid input never costs
// a database round trip. See
// internal/product/service.ProductService.Create for the identical
// reasoning applied to Products.
func (s *OLTService) Create(ctx context.Context, o olt.OLT) (olt.OLT, error) {
	if err := o.Validate(); err != nil {
		return olt.OLT{}, err
	}
	return s.olts.Create(ctx, o)
}

// Update validates o and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *OLTService) Update(ctx context.Context, o olt.OLT) (olt.OLT, error) {
	if err := o.Validate(); err != nil {
		return olt.OLT{}, err
	}
	return s.olts.Update(ctx, o)
}

// Delete removes the OLT identified by id.
func (s *OLTService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.olts.Delete(ctx, id)
}
