package ponport

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether p has every required field set: a present
// OLTID and a positive PortNumber. Description is optional and is never
// checked for presence, consistent with olt.OLT.Validate.
//
// PortNumber is required means PortNumber must be greater than zero, not
// merely "set" — Go's int has no separate "unset" state the way a
// pointer or a uuid.UUID does, and 0 is not a real-world PON port number
// under the conventional 1-based numbering this package assumes (see
// PONPort's doc comment on what this package deliberately does not
// model). Treating PortNumber <= 0 as missing is the same reasoning
// CustomerID == uuid.Nil uses elsewhere to mean "not provided" — the
// zero value doubling as "absent" because it can never be a legitimate
// value.
func (p PONPort) Validate() error {
	errs := validate.New()

	if p.OLTID == uuid.Nil {
		errs.Add("olt_id", "is required")
	}
	if p.PortNumber <= 0 {
		errs.Add("port_number", "is required")
	}

	return errs.Err()
}
