package catalog_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
)

// stubCatalogRepository has no SQL implementation to test yet — that is
// internal/catalog/postgres's job. It exists solely to prove
// CatalogRepository is satisfiable with a sane, consistent method shape,
// mirroring internal/location/repository_test.go's stub for the same
// reason: the var block's compile-time assertion is the actual check —
// this file fails to build if the interface and this stub ever drift
// apart.
type stubCatalogRepository struct{}

func (stubCatalogRepository) Get(context.Context, uuid.UUID) (catalog.ProductCatalog, error) {
	return catalog.ProductCatalog{}, nil
}
func (stubCatalogRepository) List(context.Context) ([]catalog.ProductCatalog, error) {
	return nil, nil
}
func (stubCatalogRepository) Create(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	return c, nil
}
func (stubCatalogRepository) Update(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	return c, nil
}
func (stubCatalogRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ catalog.CatalogRepository = (*stubCatalogRepository)(nil)

func TestCatalogRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, CatalogRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
