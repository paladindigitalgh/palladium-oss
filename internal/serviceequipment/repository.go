package serviceequipment

import (
	"context"

	"github.com/google/uuid"
)

// ServiceEquipmentRepository persists ServiceEquipment records. Get,
// List, Create, Update, and Delete follow the exact shape of every other
// repository in this codebase (see e.g. internal/service.ServiceRepository):
// Create and Update return the persisted entity so a caller sees anything
// the store sets (e.g. timestamps) without a second read.
//
// GetActiveByDeviceID is the one addition goal 4 asks for by name. It
// exists specifically to support the active-assignment-uniqueness
// business rule ServiceEquipmentService enforces (see
// internal/serviceequipment/service): before creating a new assignment,
// the service layer asks "does this Device already have an active one,"
// and this is the query that answers it directly, in one round trip,
// rather than every caller fetching List and filtering client-side. Like
// Get, it returns an apperror.KindNotFound error when no active
// assignment exists for deviceID — that is the expected, common case (a
// Device with no current assignment), not an exceptional one.
//
// Nothing in this package implements ServiceEquipmentRepository — no SQL,
// no migrations — so the domain has zero dependency on any storage
// technology. A concrete implementation (internal/serviceequipment/postgres)
// satisfies it.
type ServiceEquipmentRepository interface {
	Get(ctx context.Context, id uuid.UUID) (ServiceEquipment, error)
	List(ctx context.Context) ([]ServiceEquipment, error)
	Create(ctx context.Context, equipment ServiceEquipment) (ServiceEquipment, error)
	Update(ctx context.Context, equipment ServiceEquipment) (ServiceEquipment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (ServiceEquipment, error)
}
