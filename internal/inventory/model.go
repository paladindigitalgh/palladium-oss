// Package inventory models Palladium's physical inventory hierarchy:
//
//	Site -> Building -> Room -> Rack -> Device
//
// This package holds only the domain model, field validation, and
// repository interfaces — no SQL, no migrations, and no HTTP CRUD. Per
// CLAUDE.md's Core Philosophy ("Never couple inventory directly to
// customers"), nothing here references customers or services; these
// entities exist independently of who, if anyone, is being served by them.
//
// Device is intentionally generic. Vendor- and function-specific concepts
// (OLTs, routers, switches, cards, ports, splitters, fiber, ...) are out of
// scope here — see CLAUDE.md's Plugin Philosophy — and will arrive as
// later, more specific models.
package inventory

import (
	"time"

	"github.com/google/uuid"
)

// Metadata holds the fields common to every inventory entity: an identity,
// a human-readable label, free-form notes, and lifecycle timestamps.
type Metadata struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Site represents a physical location — a Central Office, POP, Hub, or
// similar — and is the root of the inventory hierarchy.
type Site struct {
	Metadata
}

// Building is a physical structure located at a Site. SiteID is required:
// a Building is defined by the location it occupies.
type Building struct {
	Metadata
	SiteID uuid.UUID
}

// Room is a space inside a Building. BuildingID is required for the same
// reason Building.SiteID is: a Room is defined by the structure it is in.
type Room struct {
	Metadata
	BuildingID uuid.UUID
}

// Rack is equipment framing that, once installed, sits inside a Room.
//
// RoomID is nullable. Per the Inventory Philosophy lifecycle described in
// docs/ARCHITECTURE.md (Ordered -> Received -> Stored -> Installed ->
// Provisioned -> Assigned -> Retired -> Disposed), a rack can exist in
// inventory before it is installed anywhere — e.g. sitting in a warehouse
// — so its Room association cannot be required.
type Rack struct {
	Metadata
	RoomID *uuid.UUID
}

// Device is any installed piece of equipment.
//
// RackID is nullable for the same reason as Rack.RoomID: a device can be
// ordered, received, and stored before it is ever racked.
//
// Manufacturer, Model, and SerialNumber are plain strings rather than
// separate Manufacturer/Model entities. Per CLAUDE.md's "avoid unnecessary
// abstractions": normalizing them into their own tables only pays off once
// something needs to query or dedupe across devices by manufacturer or
// model, and nothing does yet — see docs/ARCHITECTURE.md's Data
// Philosophy ("denormalize only after measurement proves necessary",
// applied here in reverse: don't normalize before it's needed either).
//
// AssetTag is optional, so it is a plain string (empty means "not set"),
// consistent with how Metadata.Description — also optional — is handled.
type Device struct {
	Metadata
	RackID       *uuid.UUID
	Manufacturer string
	Model        string
	SerialNumber string
	AssetTag     string
	Status       DeviceStatus
}
