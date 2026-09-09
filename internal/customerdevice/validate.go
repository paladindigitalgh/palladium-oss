package customerdevice

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether c has every required field set: a present
// CustomerID and a present DeviceID. Description is optional and never
// checked for presence, consistent with serviceequipment.ServiceEquipment.
//
// The active-assignment-uniqueness business rule ("a Device may be
// attached to at most one Customer at a time") is deliberately not
// checked here, for the exact reason
// serviceequipment.ServiceEquipment.Validate's own doc comment gives:
// Validate only ever answers "is this record well-formed in isolation,"
// never a question requiring a repository round trip. That belongs in
// CustomerDeviceService (see internal/customerdevice/service).
func (c CustomerDevice) Validate() error {
	errs := validate.New()

	if c.CustomerID == uuid.Nil {
		errs.Add("customer_id", "is required")
	}
	if c.DeviceID == uuid.Nil {
		errs.Add("device_id", "is required")
	}

	return errs.Err()
}
