package product_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

// stubProductRepository has no SQL implementation to test yet — that is
// internal/product/postgres's job. It exists solely to prove
// ProductRepository is satisfiable with a sane, consistent method shape,
// mirroring internal/catalog/repository_test.go's stub for the same
// reason: the var block's compile-time assertion is the actual check —
// this file fails to build if the interface and this stub ever drift
// apart.
type stubProductRepository struct{}

func (stubProductRepository) Get(context.Context, uuid.UUID) (product.Product, error) {
	return product.Product{}, nil
}
func (stubProductRepository) List(context.Context) ([]product.Product, error) { return nil, nil }
func (stubProductRepository) Create(_ context.Context, p product.Product) (product.Product, error) {
	return p, nil
}
func (stubProductRepository) Update(_ context.Context, p product.Product) (product.Product, error) {
	return p, nil
}
func (stubProductRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ product.ProductRepository = (*stubProductRepository)(nil)

func TestProductRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ProductRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
