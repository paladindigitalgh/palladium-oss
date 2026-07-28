package accessinterface

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether a has every required field set: a present
// PONPortID, a valid Technology, a non-empty Name, and a valid Status.
// Description is optional and is never checked for presence, consistent
// with every other domain's Validate in this codebase.
func (a AccessInterface) Validate() error {
	errs := validate.New()

	if a.PONPortID == uuid.Nil {
		errs.Add("pon_port_id", "is required")
	}
	if !a.Technology.Valid() {
		errs.Add("technology", fmt.Sprintf("must be one of: %s", technologyNames()))
	}
	if !validate.Required(a.Name) {
		errs.Add("name", "is required")
	}
	if !a.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", statusNames()))
	}

	return errs.Err()
}
