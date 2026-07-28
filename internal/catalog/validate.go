package catalog

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether c has every required field set: a present
// Name and a Status that is one of its defined values (see status.go).
// Description is optional and is never checked for presence, consistent
// with customer.Customer.Validate.
func (c ProductCatalog) Validate() error {
	errs := validate.New()

	if !validate.Required(c.Name) {
		errs.Add("name", "is required")
	}
	if !c.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", catalogStatusNames()))
	}

	return errs.Err()
}
