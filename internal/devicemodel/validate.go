package devicemodel

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether m has every required field set: a present
// Name and a non-nil ManufacturerID. A DeviceModel with no Manufacturer
// is meaningless — see model.go's own doc comment — so ManufacturerID
// is required the same way inventory.Building.SiteID is.
func (m DeviceModel) Validate() error {
	errs := validate.New()

	if !validate.Required(m.Name) {
		errs.Add("name", "is required")
	}
	if m.ManufacturerID == uuid.Nil {
		errs.Add("manufacturer_id", "is required")
	}

	return errs.Err()
}
