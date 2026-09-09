package devicemanufacturer

import (
	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether m has every required field set: a present
// Name. Description is optional and is never checked for presence,
// consistent with provider.Provider.Validate.
func (m DeviceManufacturer) Validate() error {
	errs := validate.New()

	if !validate.Required(m.Name) {
		errs.Add("name", "is required")
	}

	return errs.Err()
}
