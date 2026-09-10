package serviceequipment

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether e has every required field set: a present
// ServiceID, a present DeviceID, and a Role that is one of its defined
// values (see role.go). Description is optional and is never checked for
// presence, consistent with service.Service.Validate. UNIPort is
// required, and must be 1 or 2, only for EquipmentRoleONU/ONT -- the two
// Roles internal/provisioning/kontron/service.ServiceProfileService's own
// run method already treats as its only concern (see that method's doc
// comment); it is never checked for any other Role, since a Router,
// WiFiAccessPoint, UPS, or Gateway has no "uni" port for this to mean
// anything about.
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
	if (e.Role == EquipmentRoleONU || e.Role == EquipmentRoleONT) && e.UNIPort != 1 && e.UNIPort != 2 {
		errs.Add("uni_port", "must be 1 (10GE) or 2 (1GE) for ONU/ONT equipment")
	}

	return errs.Err()
}
