package customer

import (
	"context"

	"github.com/google/uuid"
)

// CustomerRepository persists Customers. It follows the exact shape of the
// Inventory repositories (internal/inventory/repository.go): Get, List,
// Create, Update, Delete, with Create and Update returning the persisted
// entity so a caller sees anything the store sets (e.g. timestamps)
// without a second read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/customer/postgres) satisfies it.
type CustomerRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Customer, error)
	List(ctx context.Context) ([]Customer, error)
	Create(ctx context.Context, customer Customer) (Customer, error)
	Update(ctx context.Context, customer Customer) (Customer, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
