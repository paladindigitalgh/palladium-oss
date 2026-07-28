package service

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether s has every required field set: a present
// LocationID, a present ProductID, a present ServiceProfileID, and a
// Status that is one of its defined values (see status.go). Description
// is optional and is never checked for presence, consistent with
// location.Location.Validate.
//
// ActivatedAt, SuspendedAt, and DisconnectedAt are never validated
// against Status here (e.g. requiring ActivatedAt once Status is Active).
// That would be exactly the state-machine/transition logic status.go's
// doc comment explains this milestone deliberately does not build; those
// three fields are free-form timestamps a caller may set independently of
// Status for now.
func (s Service) Validate() error {
	errs := validate.New()

	if s.LocationID == uuid.Nil {
		errs.Add("location_id", "is required")
	}
	if s.ProductID == uuid.Nil {
		errs.Add("product_id", "is required")
	}
	if s.ServiceProfileID == uuid.Nil {
		errs.Add("service_profile_id", "is required")
	}
	if !s.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", serviceStatusNames()))
	}

	return errs.Err()
}
