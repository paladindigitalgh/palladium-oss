// Package service is the Product domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/product/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/product/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/location/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

// ProductService is the Product domain's business logic.
//
// It depends only on product.ProductRepository — not clock.Clock, for the
// same reason internal/location/service.LocationService does not:
// timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now".
type ProductService struct {
	products product.ProductRepository
}

// NewProductService builds a ProductService.
func NewProductService(products product.ProductRepository) *ProductService {
	return &ProductService{products: products}
}

// Get retrieves a Product by ID.
func (s *ProductService) Get(ctx context.Context, id uuid.UUID) (product.Product, error) {
	return s.products.Get(ctx, id)
}

// List returns every Product.
func (s *ProductService) List(ctx context.Context) ([]product.Product, error) {
	return s.products.List(ctx)
}

// Create validates p and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of ProductService — so every caller
// gets the same guarantee for free, and invalid input never costs a
// database round trip. See internal/location/service.LocationService.Create
// for the identical reasoning applied to Locations.
func (s *ProductService) Create(ctx context.Context, p product.Product) (product.Product, error) {
	if err := p.Validate(); err != nil {
		return product.Product{}, err
	}
	return s.products.Create(ctx, p)
}

// Update validates p and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *ProductService) Update(ctx context.Context, p product.Product) (product.Product, error) {
	if err := p.Validate(); err != nil {
		return product.Product{}, err
	}
	return s.products.Update(ctx, p)
}

// Delete removes the Product identified by id.
func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.products.Delete(ctx, id)
}
