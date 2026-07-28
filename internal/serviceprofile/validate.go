package serviceprofile

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether p has every required field set: a present
// Name and a Status that is one of its defined values (see status.go).
// Description is optional and is never checked for presence, consistent
// with catalog.ProductCatalog.Validate.
func (p ServiceProfile) Validate() error {
	errs := validate.New()

	if !validate.Required(p.Name) {
		errs.Add("name", "is required")
	}
	if !p.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", statusNames()))
	}

	return errs.Err()
}
