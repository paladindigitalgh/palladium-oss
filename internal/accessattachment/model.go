// Package accessattachment models Palladium's Access Attachment domain
// (v1): the link between an AccessInterface (see internal/accessinterface)
// and the subscriber equipment (see internal/serviceequipment) physically
// attached to it — e.g. "this ONU is currently plugged into this
// interface." This package holds only the domain model, field
// validation, and the repository interface — no SQL, no migrations, no
// HTTP CRUD — mirroring internal/serviceequipment's own package exactly,
// which is this codebase's closest precedent: a link record between two
// sibling domains with an active-uniqueness business rule enforced in
// the service layer, not the database.
//
// This package does not import internal/accessinterface or
// internal/serviceequipment. AccessInterfaceID and ServiceEquipmentID are
// bare uuid.UUID values, not references to those packages' types: the
// foreign keys to access_interfaces(id) and service_equipment(id) are
// database concepts, enforced by internal/accessattachment/postgres and
// its migration, not Go package dependencies — the same reasoning
// internal/serviceequipment/model.go documents for why ServiceEquipment
// does not import internal/service or internal/inventory.
//
// An AccessAttachment record describes that equipment is (or was)
// plugged into an interface and when, never how the interface or the
// equipment itself is configured. Per this milestone's explicit scope,
// this package does not model VLANs, authentication, bandwidth, or
// provisioning — see internal/accessinterface/model.go's doc comment for
// the same exclusions one layer down.
package accessattachment

import (
	"time"

	"github.com/google/uuid"
)

// AccessAttachment links an AccessInterface to the ServiceEquipment
// attached to it.
//
// InstalledAt and RemovedAt are each *time.Time, not time.Time, for the
// same reason serviceequipment.ServiceEquipment's fields of the same
// names are: a newly created attachment may not yet record when it was
// installed, and RemovedAt in particular carries meaning beyond "a
// timestamp" — RemovedAt == nil is this milestone's literal definition of
// "active" (see the Active method's doc comment and
// internal/accessattachment/service's uniqueness rule). The zero value
// of time.Time is a real (if nonsensical) instant, so it cannot double
// as "not yet removed" without risking a genuine 0001-01-01 timestamp
// being read as a removal.
type AccessAttachment struct {
	ID                 uuid.UUID
	AccessInterfaceID  uuid.UUID
	ServiceEquipmentID uuid.UUID
	InstalledAt        *time.Time
	RemovedAt          *time.Time
	RemovalReason      string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Active reports whether this attachment is currently in effect: it has
// not been removed. This is the exact predicate this milestone's goal 2
// defines ("Active means: RemovedAt == nil") and the one
// internal/accessattachment/service's uniqueness rule uses — defined
// once here, on the domain type itself, so the repository, the service
// layer, and any future caller all agree on what "active" means without
// each re-deriving the same nil check. Mirrors
// serviceequipment.ServiceEquipment.Active exactly.
func (a AccessAttachment) Active() bool {
	return a.RemovedAt == nil
}
