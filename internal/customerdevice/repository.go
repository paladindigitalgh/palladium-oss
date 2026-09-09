package customerdevice

import (
	"context"

	"github.com/google/uuid"
)

// CustomerDeviceRepository persists CustomerDevice records. Get, List,
// Create, and Update follow the exact shape of every other repository in
// this codebase (see e.g. internal/serviceequipment.ServiceEquipmentRepository):
// Create and Update return the persisted entity so a caller sees anything
// the store sets (e.g. timestamps) without a second read.
//
// There is deliberately no Delete: CustomerDeviceService's Update is how
// a Device is detached (DetachedAt set, the row itself preserved) — the
// same "always preserve history" decision this codebase already made
// explicit for inventory.Device itself (see that package's own
// DeviceRepository doc comment). A CustomerDevice row is never erased.
//
// GetActiveByDeviceID exists specifically to support the
// active-assignment-uniqueness business rule CustomerDeviceService
// enforces: before attaching a Device to a Customer, the service layer
// asks "is this Device already attached somewhere," and this is the
// query that answers it directly, in one round trip — the same
// reasoning serviceequipment.ServiceEquipmentRepository's own
// GetActiveByDeviceID doc comment gives. Like Get, it returns an
// apperror.KindNotFound error when no active attachment exists for
// deviceID — the expected, common case, not an exceptional one.
//
// Nothing in this package implements CustomerDeviceRepository — no SQL,
// no migrations — so the domain has zero dependency on any storage
// technology. A concrete implementation (internal/customerdevice/postgres)
// satisfies it.
type CustomerDeviceRepository interface {
	Get(ctx context.Context, id uuid.UUID) (CustomerDevice, error)
	List(ctx context.Context) ([]CustomerDevice, error)
	Create(ctx context.Context, cd CustomerDevice) (CustomerDevice, error)
	Update(ctx context.Context, cd CustomerDevice) (CustomerDevice, error)
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (CustomerDevice, error)
}
