package customer

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether c has every required field set: a present
// Name, and a CustomerType and CustomerStatus that are each one of their
// defined values (see customer_type.go and status.go). Description is
// optional and is never checked for presence, consistent with how
// inventory.Metadata.Description is handled.
func (c Customer) Validate() error {
	errs := validate.New()

	if !validate.Required(c.Name) {
		errs.Add("name", "is required")
	}
	if !c.CustomerType.Valid() {
		errs.Add("customer_type", fmt.Sprintf("must be one of: %s", customerTypeNames()))
	}
	if !c.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", customerStatusNames()))
	}

	return errs.Err()
}
