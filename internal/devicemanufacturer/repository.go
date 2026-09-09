package devicemanufacturer

import (
	"context"

	"github.com/google/uuid"
)

// DeviceManufacturerRepository persists DeviceManufacturers. It follows
// the exact shape of every other repository in this codebase (see e.g.
// internal/oltmodel.OLTModelRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a
// second read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/devicemanufacturer/postgres) satisfies it.
type DeviceManufacturerRepository interface {
	Get(ctx context.Context, id uuid.UUID) (DeviceManufacturer, error)
	List(ctx context.Context) ([]DeviceManufacturer, error)
	Create(ctx context.Context, m DeviceManufacturer) (DeviceManufacturer, error)
	Update(ctx context.Context, m DeviceManufacturer) (DeviceManufacturer, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
