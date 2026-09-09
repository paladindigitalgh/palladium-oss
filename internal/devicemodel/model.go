// Package devicemodel models Palladium's Device Model domain (v1): an
// Administration-managed catalog entry naming one hardware model (e.g.
// "G-140W-CT") made by a internal/devicemanufacturer.DeviceManufacturer
// (e.g. "Nokia") — the pair inventory.Device's DeviceModelID references
// in place of what used to be two free-text Manufacturer/Model string
// fields directly on Device (see internal/inventory/model.go's Device
// doc comment for why that changed). This package holds only the domain
// model, field validation, and the repository interface — no SQL, no
// migrations, no HTTP CRUD — mirroring internal/oltmodel's own package
// exactly, one level down: a DeviceModel is to a DeviceManufacturer what
// an OLTModel is to its Vendor, except Vendor is a closed enum on the
// catalog entry itself while ManufacturerID here is a reference to its
// own catalog, because device manufacturers are not a small closed set
// (see internal/devicemanufacturer's own doc comment).
package devicemodel

import (
	"time"

	"github.com/google/uuid"
)

// DeviceModel is a catalog entry naming one hardware model, made by the
// DeviceManufacturer identified by ManufacturerID.
type DeviceModel struct {
	ID             uuid.UUID
	ManufacturerID uuid.UUID
	Name           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
