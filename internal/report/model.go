// Package report backs Explorer (docs/09-WORKSPACE-SPECIFICATIONS.md
// §15): Palladium's curated set of cross-domain, read-only reports.
//
// Unlike every other domain package in this codebase, a "report" has no
// single owning table and no lifecycle of its own — each one is a real,
// hand-written SQL join across other domains' tables (see
// internal/report/postgres), read-only, with no Create/Update/Delete.
// This package has no dependency on customer, inventory, service, or any
// other domain's Go types: each row struct below is its own flat,
// already-joined shape, not a reuse of e.g. customer.Customer, the same
// "a report row is its own thing, not a repurposed domain type" reasoning
// that keeps this package decoupled from every domain its SQL reads from.
package report

import "github.com/google/uuid"

// CustomerContactRow is one row of the "Customers & Contacts" report:
// every Customer, with one row per Contact they have on file. A Customer
// with no Contacts still appears exactly once, with every Contact field
// blank — this is a report meant to answer "who do I call for every
// customer," so a customer who happens to have no contact on file must
// still show up, not silently vanish from a join.
type CustomerContactRow struct {
	CustomerID     uuid.UUID
	CustomerName   string
	CustomerType   string
	CustomerStatus string

	// ContactID is uuid.Nil when this Customer has no Contact at all
	// (see the type's own doc comment above) — every other Contact field
	// is blank in that case too.
	ContactID     uuid.UUID
	ContactName   string
	ContactRole   string
	ContactEmail  string
	ContactPhone  string
	ContactStatus string
}

// DeviceRow is one row of the "Devices" report: every Device, with its
// resolved Manufacturer/Model names (see devicemanufacturer/devicemodel),
// its physical location path if it is racked, and whichever Customer
// currently has it, if any.
//
// AssignedCustomerName resolves through whichever of the two paths
// docs/03-DOMAIN-MODEL.md §9 describes is currently active for this
// Device: a direct CustomerDevice placement, or an active
// ServiceEquipment assignment (via its Service's Location). A Device is
// not expected to have both active at once in practice, but if it
// somehow did, the query picks one rather than duplicating the row — see
// postgres/report.go's own comment on the LATERAL join for exactly which
// one wins.
type DeviceRow struct {
	DeviceID     uuid.UUID
	DeviceName   string
	SerialNumber string
	AssetTag     string
	DeviceStatus string
	Manufacturer string
	Model        string

	// SiteName/BuildingName/RoomName/RackName are all empty when the
	// Device is not racked (see inventory.Device's own RackID, nullable
	// for the same "ordered, received, stored before it is ever racked"
	// reason racks.room_id is nullable).
	SiteName     string
	BuildingName string
	RoomName     string
	RackName     string

	// AssignedCustomerID is uuid.Nil, and AssignedCustomerName empty,
	// when no Customer currently has this Device through either path.
	AssignedCustomerID   uuid.UUID
	AssignedCustomerName string
}

// CustomerDeviceRow is one row of the "Customers & Devices" report:
// every currently-active Customer-Device relationship, covering both
// paths docs/03-DOMAIN-MODEL.md §9 describes a Device can reach a
// Customer through. Unlike DeviceRow's single "whichever one wins"
// column, this report is the other direction: it never collapses the
// two paths into one row per Device — a Device delivered by more than
// one active relationship (unusual, but not disallowed) gets one row
// per relationship, exactly like a real join would.
type CustomerDeviceRow struct {
	CustomerID     uuid.UUID
	CustomerName   string
	CustomerType   string
	CustomerStatus string

	DeviceID     uuid.UUID
	DeviceName   string
	SerialNumber string
	Manufacturer string
	Model        string
	DeviceStatus string

	// Relationship is "Placement" (a CustomerDevice record) or "Service
	// Equipment" (a ServiceEquipment assignment) — see this type's own
	// doc comment above.
	Relationship string

	// ServiceStatus/LocationName are only ever set on a "Service
	// Equipment" row — a "Placement" row leaves both blank, since a
	// CustomerDevice placement has no Service or Location of its own.
	ServiceStatus string
	LocationName  string
}
