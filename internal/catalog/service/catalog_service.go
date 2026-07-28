// Package service is the Catalog domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/catalog/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/catalog/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/location/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
)

// CatalogService is the Catalog domain's business logic.
//
// It depends only on catalog.CatalogRepository — not clock.Clock, for the
// same reason internal/location/service.LocationService does not:
// timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now".
type CatalogService struct {
	catalogs catalog.CatalogRepository
}

// NewCatalogService builds a CatalogService.
func NewCatalogService(catalogs catalog.CatalogRepository) *CatalogService {
	return &CatalogService{catalogs: catalogs}
}

// Get retrieves a ProductCatalog by ID.
func (s *CatalogService) Get(ctx context.Context, id uuid.UUID) (catalog.ProductCatalog, error) {
	return s.catalogs.Get(ctx, id)
}

// List returns every ProductCatalog.
func (s *CatalogService) List(ctx context.Context) ([]catalog.ProductCatalog, error) {
	return s.catalogs.List(ctx)
}

// Create validates c and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of CatalogService — so every caller
// gets the same guarantee for free, and invalid input never costs a
// database round trip. See internal/location/service.LocationService.Create
// for the identical reasoning applied to Locations.
func (s *CatalogService) Create(ctx context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	if err := c.Validate(); err != nil {
		return catalog.ProductCatalog{}, err
	}
	return s.catalogs.Create(ctx, c)
}

// Update validates c and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *CatalogService) Update(ctx context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	if err := c.Validate(); err != nil {
		return catalog.ProductCatalog{}, err
	}
	return s.catalogs.Update(ctx, c)
}

// Delete removes the ProductCatalog identified by id.
func (s *CatalogService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.catalogs.Delete(ctx, id)
}
