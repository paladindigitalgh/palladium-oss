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
// ListByCustomerID is this package's one addition beyond that standard
// shape, added for internal/accesstopology: resolving where a Customer's
// equipment sits on the access network starts by finding every Location
// (service address) that Customer has, and no existing method answers
// that directly. It follows serviceequipment.ServiceEquipmentRepository.
// ListActiveByServiceID's own precedent — a query shaped around one
// specific, already-real need, rather than making every caller fetch
// List and filter client-side for "customer_id = X". Unlike that method,
// this one has no "active" filter of its own: Location carries no
// removed/active-uniqueness concept the way ServiceEquipment and
// AccessAttachment do (see location/model.go), so every Location a
// Customer has is returned, active or not — a future caller filtering by
// Location.Status is free to, but this method does not decide that for
// them.
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
	ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]Location, error)
}
