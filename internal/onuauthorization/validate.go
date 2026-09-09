package onuauthorization

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether a has every required field set: a present
// DeviceID, a present OLTID, a non-empty Interface, and a non-zero
// AuthorizedAt. DeauthorizedAt is optional and never checked for
// presence, consistent with every other RemovedAt-shaped field in this
// codebase (see model.go's own doc comment on OnuAuthorization.Active).
func (a OnuAuthorization) Validate() error {
	errs := validate.New()

	if a.DeviceID == uuid.Nil {
		errs.Add("device_id", "is required")
	}
	if a.OLTID == uuid.Nil {
		errs.Add("olt_id", "is required")
	}
	if !validate.Required(a.Interface) {
		errs.Add("interface", "is required")
	}
	if a.AuthorizedAt.IsZero() {
		errs.Add("authorized_at", "is required")
	}

	return errs.Err()
}
