package accessattachment

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether a has every required field set: a present
// AccessInterfaceID and a present ServiceEquipmentID. RemovalReason,
// InstalledAt, and RemovedAt are all optional and never checked for
// presence, consistent with serviceequipment.ServiceEquipment.Validate.
//
// The active-attachment-uniqueness business rule (goal 2: "a
// ServiceEquipment record may have only one active Access Attachment")
// is deliberately not checked here, for the exact reason
// serviceequipment.ServiceEquipment.Validate's doc comment gives for its
// own uniqueness rule: Validate only ever answers "is this record
// well-formed in isolation," never "does this conflict with anything
// else already persisted" — that requires a repository round-trip, which
// belongs in AccessAttachmentService (see
// internal/accessattachment/service), the layer that already owns the
// repository dependency Validate deliberately does not have.
func (a AccessAttachment) Validate() error {
	errs := validate.New()

	if a.AccessInterfaceID == uuid.Nil {
		errs.Add("access_interface_id", "is required")
	}
	if a.ServiceEquipmentID == uuid.Nil {
		errs.Add("service_equipment_id", "is required")
	}

	return errs.Err()
}
