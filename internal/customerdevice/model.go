// Package customerdevice models Palladium's Customer Device domain (v1):
// the link between a customer.Customer and the inventory.Device
// physically placed at one of their premises — e.g. "this ONT is
// sitting in this customer's living room" — independent of whether any
// service.Service has been set up to use it yet. It holds only the
// domain model, field validation, and the repository interface — no
// SQL, no migrations, no HTTP CRUD — mirroring internal/serviceequipment's
// own package exactly, one level up: where a ServiceEquipment record
// answers "which Device is delivering this Service," a CustomerDevice
// record answers "which Device is at this Customer's premises at all,"
// a question that has to be answerable before a Service exists to ask
// it about.
//
// This package does not import internal/customer or internal/inventory.
// CustomerID and DeviceID are bare uuid.UUID values, not references to
// customer.Customer or inventory.Device: the foreign keys to
// customers(id) and devices(id) are database concepts, enforced by
// internal/customerdevice/postgres and its migration, not Go package
// dependencies — the same reasoning internal/serviceequipment/model.go
// documents for its own ServiceID/DeviceID.
//
// CLAUDE.md's Core Philosophy says "never couple inventory directly to
// customers" — this package is the one deliberate, narrow exception,
// added because real installs put a physical Device at a Customer's
// premises before, or entirely without, an active Service (a truck roll
// that racks an ONT days before the account goes live). It couples the
// same way internal/serviceequipment already does one domain over
// (Service to Device): a join recording that a relationship exists and
// when, never a field on Device itself, and never anything Device's own
// package (internal/inventory) knows about.
package customerdevice

import (
	"time"

	"github.com/google/uuid"
)

// CustomerDevice links a Customer to a Device placed at their premises.
//
// AttachedAt and DetachedAt are each *time.Time, not time.Time, for the
// same reason serviceequipment.ServiceEquipment's InstalledAt/RemovedAt
// are: a newly created attachment may not yet record when it was
// attached, and DetachedAt in particular carries meaning beyond "a
// timestamp" — DetachedAt == nil is this package's literal definition of
// "active" (see Active and internal/customerdevice/service's
// uniqueness rule). The zero value of time.Time is a real (if
// nonsensical) instant, so it cannot double as "not yet detached"
// without risking a genuine 0001-01-01 timestamp being read as a
// detachment.
type CustomerDevice struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	DeviceID    uuid.UUID
	Description string

	AttachedAt *time.Time
	DetachedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Active reports whether this attachment is currently in effect: the
// Device has not been detached. This is the same predicate
// serviceequipment.ServiceEquipment.Active defines for its own
// RemovedAt, applied here to DetachedAt — defined once, on the domain
// type itself, so the repository, the service layer, and any future
// caller all agree on what "active" means without each re-deriving the
// same nil check.
func (c CustomerDevice) Active() bool {
	return c.DetachedAt == nil
}
