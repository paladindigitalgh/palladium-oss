package contact

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether c has every required field set: a present
// CustomerID, a present Name, and a Role and Status that are each one of
// their defined values (see contact_role.go and status.go).
//
// Email, Phone, and Description are never checked — the same "optional,
// plain string" treatment location.Location's address fields get, and
// for the same reason: not every Contact necessarily has both an email
// and a phone number on file.
func (c Contact) Validate() error {
	errs := validate.New()

	if c.CustomerID == uuid.Nil {
		errs.Add("customer_id", "is required")
	}
	if !validate.Required(c.Name) {
		errs.Add("name", "is required")
	}
	if !c.Role.Valid() {
		errs.Add("role", fmt.Sprintf("must be one of: %s", contactRoleNames()))
	}
	if !c.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", contactStatusNames()))
	}

	return errs.Err()
}
