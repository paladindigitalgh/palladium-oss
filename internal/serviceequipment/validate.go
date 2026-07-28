package serviceequipment

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether e has every required field set: a present
// ServiceID, a present DeviceID, and a Role that is one of its defined
// values (see role.go). Description is optional and is never checked for
// presence, consistent with service.Service.Validate.
//
// The active-assignment-uniqueness business rule (goal 2: "a device may
// have only one active assignment") is deliberately not checked here.
// Validate only ever answers "is this record well-formed in isolation" —
// every domain package's Validate in this codebase does the same, and
// none of them queries a repository to do it. Uniqueness requires
// comparing against other persisted rows, which is a repository
// round-trip; that belongs in ServiceEquipmentService (see
// internal/serviceequipment/service), the layer that already owns the
// repository dependency Validate deliberately does not have.
func (e ServiceEquipment) Validate() error {
	errs := validate.New()

	if e.ServiceID == uuid.Nil {
		errs.Add("service_id", "is required")
	}
	if e.DeviceID == uuid.Nil {
		errs.Add("device_id", "is required")
	}
	if !e.Role.Valid() {
		errs.Add("role", fmt.Sprintf("must be one of: %s", equipmentRoleNames()))
	}

	return errs.Err()
}
