// Package serviceequipment models Palladium's Service Equipment domain
// (v1): the link between a subscriber's purchased Service (see
// internal/service) and the physical inventory.Device (see
// internal/inventory) actually delivering it — e.g. "this ONU is
// currently installed for this Service." This package holds only the
// domain model, field validation, and the repository interface — no SQL,
// no migrations, no HTTP CRUD — mirroring internal/service's own package
// exactly.
//
// This package does not import internal/service or internal/inventory.
// ServiceID and DeviceID are bare uuid.UUID values, not references to
// service.Service or inventory.Device: the foreign keys to services(id)
// and devices(id) are database concepts, enforced by
// internal/serviceequipment/postgres and its migration, not Go package
// dependencies — the same reasoning internal/service/model.go documents
// for why Service does not import internal/location or internal/product.
//
// A ServiceEquipment record describes that a link exists and what role
// the Device plays, never how the Device is configured to deliver it.
// Per this milestone's explicit scope:
//
//   - No provisioning, no firmware, no OLT, no VLAN, no IP addresses, no
//     automation: nothing here configures the Device or touches the
//     network. A future Provisioning domain (see
//     docs/ARCHITECTURE.md's Provisioning and Network domains) will act
//     on a Device because of a ServiceEquipment record, not the other
//     way around.
//
// UNIPort (added 2026-09-10) is the one narrow, deliberate exception to
// that "never how it is configured" rule -- mirroring how
// internal/customerdevice became the one deliberate exception to "never
// couple inventory directly to customers" (see CLAUDE.md's Core
// Philosophy and docs/03-DOMAIN-MODEL.md section 26). Real Kontron
// provisioning needs to know which physical LAN port on the ONU/ONT a
// Service is delivered over -- the OLT command is literally
// "service-profile <profile> uni <N>" -- and that choice has to persist
// somewhere every future re-provisioning attempt (Suspend, Resume, a
// manual retry) can read it back from, not just a one-off argument to
// the workflow that first applies it. ServiceEquipment is the one record
// that already names both the Service and the Device, so it is where
// that answer lives, not a new sibling domain that would just duplicate
// this join. It stays a plain, vendor-agnostic int (1 or 2), never a
// literal "uni 1"/Kontron CLI fragment -- constructing the actual
// command string is internal/provisioning/kontron's job alone (see
// CLAUDE.md's Plugin Philosophy).
//
// This is the first domain in this codebase whose business logic layer
// enforces a rule beyond "is every required field present and valid":
// see internal/serviceequipment/service's doc comment for the
// active-assignment-uniqueness rule this milestone specifically asks for.
package serviceequipment

import (
	"time"

	"github.com/google/uuid"
)

// ServiceEquipment links a Service to the Device fulfilling it.
//
// InstalledAt and RemovedAt are each *time.Time, not time.Time, for the
// same reason service.Service's ActivatedAt/SuspendedAt/DisconnectedAt
// are: a newly created assignment may not yet record when it was
// installed, and RemovedAt in particular carries meaning beyond "a
// timestamp" — RemovedAt == nil is this milestone's literal definition of
// "active" (see the Role and Validate doc comments, and
// internal/serviceequipment/service's uniqueness rule). The zero value of
// time.Time is a real (if nonsensical) instant, so it cannot double as
// "not yet removed" without risking a genuine 0001-01-01 timestamp being
// read as a removal.
type ServiceEquipment struct {
	ID          uuid.UUID
	ServiceID   uuid.UUID
	DeviceID    uuid.UUID
	Role        EquipmentRole
	Description string

	// UNIPort is 1 ("10GE") or 2 ("1GE") for ONU/ONT equipment -- which
	// physical LAN port on the Device this Service is delivered over
	// (see this type's own doc comment). Meaningless, and never
	// validated, for any other Role.
	UNIPort int

	InstalledAt *time.Time
	RemovedAt   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Active reports whether this assignment is currently in effect: it has
// not been removed. This is the exact predicate goal 2 defines ("Active
// means: RemovedAt == nil") and the one
// internal/serviceequipment/service's uniqueness rule uses — defined once
// here, on the domain type itself, so the repository, the service layer,
// and any future caller all agree on what "active" means without each
// re-deriving the same nil check.
func (e ServiceEquipment) Active() bool {
	return e.RemovedAt == nil
}
