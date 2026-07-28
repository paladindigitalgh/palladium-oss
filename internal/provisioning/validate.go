package provisioning

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether j has every required field set: a present
// ServiceID, and an Operation and Status that are each one of their
// defined values (see operation.go and status.go).
//
// This deliberately never checks whether j.Status is a legal transition
// from whatever j's previous persisted Status was — goal 4 is explicit
// that Validate must remain a pure function, and answering "is this a
// legal transition" requires knowing the current persisted state, which
// only a repository round-trip can provide. That check belongs to
// internal/provisioning/service.ProvisioningService, the layer that
// already holds the repository dependency needed to make it (see
// ProvisioningStatus.CanTransitionTo's own doc comment for the same
// division of responsibility, one level down).
//
// RequestedByUserID, ErrorMessage, StartedAt, and CompletedAt are all
// optional and are never checked for presence — each is meaningful only
// at specific points in a job's lifecycle (see ProvisioningJob's doc
// comment), not on every valid job.
func (j ProvisioningJob) Validate() error {
	errs := validate.New()

	if j.ServiceID == uuid.Nil {
		errs.Add("service_id", "is required")
	}
	if !j.Operation.Valid() {
		errs.Add("operation", fmt.Sprintf("must be one of: %s", provisioningOperationNames()))
	}
	if !j.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", provisioningStatusNames()))
	}

	return errs.Err()
}
