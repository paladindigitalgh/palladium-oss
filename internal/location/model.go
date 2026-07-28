// Package location models Palladium's Location domain (v1): a physical
// place associated with a Customer — a service address, a billing
// address, an office, a warehouse, a POP, a data center. This package
// holds only the domain model, field validation, and the repository
// interface — no SQL, no migrations, no HTTP CRUD — mirroring
// internal/customer and internal/inventory exactly.
//
// This is deliberately not a GIS system and not a service provisioning
// system. Per this milestone's explicit scope:
//
//   - No contacts: "who do we call at this address" is a future domain,
//     not a field bolted onto this one — the same reasoning
//     internal/customer's package doc comment already gives for why it
//     has no contacts of its own.
//   - No services, no provisioning: this package cannot answer "what is
//     delivered to this Location" at all. A future Service (a later
//     phase) will reference a Location, not the other way around — see
//     docs/ARCHITECTURE.md's Customer Philosophy applied one level down.
//   - No fiber, no GIS, no maps, no geocoding: Latitude and Longitude are
//     plain, optional coordinates a caller supplies — nothing here
//     validates them against a map, looks them up from an address, or
//     understands distance, routing, or geometry. A real GIS/mapping
//     integration is a future, much larger feature.
//
// A Location references a Customer by ID only (CustomerID uuid.UUID) —
// this package does not import internal/customer at all. That is a
// deliberate decoupling, not an oversight: a foreign key is a database
// constraint, not a Go dependency, and internal/customer has no reason to
// know Locations exist. Only the PostgreSQL implementation
// (internal/location/postgres) and its tests need the real customers
// table to exist; the domain model itself only needs an identifier.
package location

import (
	"time"

	"github.com/google/uuid"
)

// Location is a physical place associated with a Customer.
//
// Latitude and Longitude are *float64, not float64, even though every
// other optional field in this codebase (Description, the address
// fields) is a plain string defaulting to "". A string's zero value ("")
// unambiguously means "not set". A float64's zero value (0) does not:
// (0, 0) is a real, meaningful coordinate — off the coast of West
// Africa, but real — so 0 cannot double as "no coordinate supplied"
// without silently corrupting data for the (admittedly rare) location
// that legitimately has one. A pointer is the only way to represent
// "unset" without also claiming a real place.
type Location struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Name       string
	Type       LocationType
	Status     LocationStatus

	Address1   string
	Address2   string
	City       string
	State      string
	PostalCode string
	Country    string

	Latitude  *float64
	Longitude *float64

	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
