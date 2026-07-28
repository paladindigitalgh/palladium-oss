package location

import (
	"context"

	"github.com/google/uuid"
)

// LocationRepository persists Locations. It follows the exact shape of
// the Inventory and Customer repositories: Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/location/postgres) satisfies it.
type LocationRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Location, error)
	List(ctx context.Context) ([]Location, error)
	Create(ctx context.Context, location Location) (Location, error)
	Update(ctx context.Context, location Location) (Location, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
