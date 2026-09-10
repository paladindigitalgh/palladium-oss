// Package devicemanufacturer models Palladium's Device Manufacturer
// domain (v1): an Administration-managed catalog entry naming a
// manufacturer of inventory.Device hardware — e.g. "Nokia" or
// "TP-Link" — that internal/devicemodel's DeviceModel entries belong to.
// This package holds only the domain model, field validation, and the
// repository interface — no SQL, no migrations, no HTTP CRUD —
// mirroring internal/provider's own package exactly.
//
// Device manufacturers are not a small closed set the way OLT vendors
// are (see internal/oltmodel.Vendor's own doc comment): inventory.Device
// covers any piece of installed equipment, from ONTs to CPE routers to
// switches, spanning far more brands than the handful of OLT chassis
// vendors this codebase's plugins integrate with. So unlike
// oltmodel.OLTModel.Vendor, Name here is free text an operator curates
// once via Administration, not a closed enum — the same reasoning
// provider.Provider.Name is free text.
//
// This package deliberately does not model manufacturer-specific
// behavior of any kind (per CLAUDE.md's Plugin Philosophy): a
// DeviceManufacturer is exactly as inert as inventory.Device.Manufacturer
// used to be as a plain string (see internal/inventory/model.go's Device
// doc comment on why that changed) — nothing here, or in
// internal/devicemodel, ever branches on which manufacturer or model a
// Device has.
package devicemanufacturer

import (
	"time"

	"github.com/google/uuid"
)

// DeviceManufacturer is a catalog entry naming one manufacturer of
// Device hardware.
//
// IsDefault marks the one DeviceManufacturer (at most) New Device's
// Manufacturer picker pre-selects (frontend DeviceFormDialog.vue) --
// still fully overridable per-Device, this only changes what the form
// starts on. It is never set through Create or the general Update (see
// this domain's own repository/service doc comments): only
// DeviceManufacturerRepository.SetDefault touches this field, which also
// clears it on whichever other DeviceManufacturer previously held it, so
// at most one is ever true (see database/migrations/00042's partial
// unique index for the same invariant enforced at the storage layer
// too).
type DeviceManufacturer struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
