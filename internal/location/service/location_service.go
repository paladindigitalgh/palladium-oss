// Package service is the Location domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/location/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/location/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/customer/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
)

// LocationService is the Location domain's business logic.
//
// It depends only on location.LocationRepository — not clock.Clock, for
// the same reason internal/customer/service.CustomerService does not:
// timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now".
type LocationService struct {
	locations location.LocationRepository
}

// NewLocationService builds a LocationService.
func NewLocationService(locations location.LocationRepository) *LocationService {
	return &LocationService{locations: locations}
}

// Get retrieves a Location by ID.
func (s *LocationService) Get(ctx context.Context, id uuid.UUID) (location.Location, error) {
	return s.locations.Get(ctx, id)
}

// List returns every Location.
func (s *LocationService) List(ctx context.Context) ([]location.Location, error) {
	return s.locations.List(ctx)
}

// Create validates l and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of LocationService — so every caller
// gets the same guarantee for free, and invalid input never costs a
// database round trip. See internal/customer/service.CustomerService.Create
// for the identical reasoning applied to Customers.
func (s *LocationService) Create(ctx context.Context, l location.Location) (location.Location, error) {
	if err := l.Validate(); err != nil {
		return location.Location{}, err
	}
	return s.locations.Create(ctx, l)
}

// Update validates l and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *LocationService) Update(ctx context.Context, l location.Location) (location.Location, error) {
	if err := l.Validate(); err != nil {
		return location.Location{}, err
	}
	return s.locations.Update(ctx, l)
}

// Delete removes the Location identified by id.
func (s *LocationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.locations.Delete(ctx, id)
}
