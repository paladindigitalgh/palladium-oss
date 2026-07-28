package catalog

import (
	"context"

	"github.com/google/uuid"
)

// CatalogRepository persists ProductCatalogs. It follows the exact shape
// of every other repository in this codebase (see e.g.
// internal/customer.CustomerRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/catalog/postgres) satisfies it.
type CatalogRepository interface {
	Get(ctx context.Context, id uuid.UUID) (ProductCatalog, error)
	List(ctx context.Context) ([]ProductCatalog, error)
	Create(ctx context.Context, catalog ProductCatalog) (ProductCatalog, error)
	Update(ctx context.Context, catalog ProductCatalog) (ProductCatalog, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
