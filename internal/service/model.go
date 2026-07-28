// Package service models Palladium's Service domain (v1): a subscriber's
// purchased Service — that a given Location has bought a given Product,
// and where that purchase currently sits in its lifecycle. This package
// holds only the domain model, field validation, and the repository
// interface — no SQL, no migrations, no HTTP CRUD — mirroring
// internal/location's own package exactly.
//
// This package does not import internal/location or internal/product.
// LocationID and ProductID are bare uuid.UUID values, not references to
// location.Location or product.Product: the foreign keys to locations(id)
// and products(id) are database concepts, enforced by
// internal/service/postgres and its migration, not Go package
// dependencies — the same reasoning internal/location/model.go documents
// for why Location does not import internal/customer.
//
// There is deliberately no CustomerID field. Per this milestone's explicit
// instruction, the customer relationship is obtained through the
// Location: a Location already has a required CustomerID (see
// internal/location.Location), so a Service naming a Customer directly
// would be a second, redundant path to the same fact — one that could
// drift from the Location's own CustomerID if the Location were ever
// reassigned to a different Customer. Looking up "which Customer owns
// this Service" is a join through Location, not a field on Service.
//
// A Service describes that a purchase exists and what state it is in,
// never how it is delivered. Per this milestone's explicit scope:
//
//   - No provisioning, no ONU, no OLT, no VLAN, no IP address, no
//     bandwidth profile, no automation: nothing here configures a device
//     or a network. A future Provisioning domain (see
//     docs/ARCHITECTURE.md's Provisioning and Network domains) will act
//     on a Service, not the other way around.
//   - No inventory assignment: a Service never references
//     internal/inventory.Device or any other Inventory type. Per
//     CLAUDE.md's Core Philosophy ("Services consume Resources. Resources
//     exist independently of Customers.") that consumption relationship
//     is a real future feature, not implied by anything here — this
//     package only records that a purchase exists, not what it consumes.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package service

import (
	"time"

	"github.com/google/uuid"
)

// Service is a subscriber's purchased Service: a Location's purchase of a
// Product, tracked through a lifecycle (see ServiceStatus in status.go).
//
// ActivatedAt, SuspendedAt, and DisconnectedAt are each *time.Time, not
// time.Time, because a Service does not pass through all of them, and
// even the ones it does pass through happen at different, independently
// meaningful moments: a newly created Service in ServiceStatusPending has
// none of them set, and the zero value of time.Time is a real (if
// unlikely) instant, so it cannot double as "this transition has not
// happened yet" without risking a genuine 0001-01-01 timestamp being
// mistaken for "never activated." This mirrors the reasoning
// internal/location/model.go gives for Latitude/Longitude being
// *float64.
type Service struct {
	ID          uuid.UUID
	LocationID  uuid.UUID
	ProductID   uuid.UUID
	Status      ServiceStatus
	Description string

	ActivatedAt    *time.Time
	SuspendedAt    *time.Time
	DisconnectedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
