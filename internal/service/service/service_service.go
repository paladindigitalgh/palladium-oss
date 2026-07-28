// Package service is the Service domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/service/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/service/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/location/service exactly.
//
// This package's import path, internal/service/service, nests a package
// named "service" inside a domain package also named "service"
// (internal/service, home to the Service struct itself). That is not a
// naming mistake: it is the exact mechanical extension of the same
// convention every other domain in this codebase already follows — the
// business logic layer for internal/location lives at
// internal/location/service, for internal/catalog at
// internal/catalog/service, and so on. The Service domain's business
// logic layer landing at internal/service/service is what applying that
// convention consistently produces; special-casing it to avoid the
// repeated word would be the actual inconsistency.
package service

import (
	"context"

	"github.com/google/uuid"

	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
)

// ServiceService is the Service domain's business logic.
//
// It depends only on domainservice.ServiceRepository — not clock.Clock,
// for the same reason internal/location/service.LocationService does
// not: timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now".
type ServiceService struct {
	services domainservice.ServiceRepository
}

// NewServiceService builds a ServiceService.
func NewServiceService(services domainservice.ServiceRepository) *ServiceService {
	return &ServiceService{services: services}
}

// Get retrieves a Service by ID.
func (s *ServiceService) Get(ctx context.Context, id uuid.UUID) (domainservice.Service, error) {
	return s.services.Get(ctx, id)
}

// List returns every Service.
func (s *ServiceService) List(ctx context.Context) ([]domainservice.Service, error) {
	return s.services.List(ctx)
}

// Create validates svc and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of ServiceService — so every caller
// gets the same guarantee for free, and invalid input never costs a
// database round trip. See internal/location/service.LocationService.Create
// for the identical reasoning applied to Locations.
func (s *ServiceService) Create(ctx context.Context, svc domainservice.Service) (domainservice.Service, error) {
	if err := svc.Validate(); err != nil {
		return domainservice.Service{}, err
	}
	return s.services.Create(ctx, svc)
}

// Update validates svc and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *ServiceService) Update(ctx context.Context, svc domainservice.Service) (domainservice.Service, error) {
	if err := svc.Validate(); err != nil {
		return domainservice.Service{}, err
	}
	return s.services.Update(ctx, svc)
}

// Delete removes the Service identified by id.
func (s *ServiceService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.services.Delete(ctx, id)
}
