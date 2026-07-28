package accessnetwork

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether a has every required field set: a present
// Name and a Status that is one of its defined values (see status.go).
// Description is optional and is never checked for presence, consistent
// with catalog.ProductCatalog.Validate.
func (a AccessNetwork) Validate() error {
	errs := validate.New()

	if !validate.Required(a.Name) {
		errs.Add("name", "is required")
	}
	if !a.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", accessNetworkStatusNames()))
	}

	return errs.Err()
}
