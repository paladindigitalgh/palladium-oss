package olt

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether o has every required field set: a present
// Name and a present OLTModelID. ManagementIPAddress and Description are
// both optional and are never checked for presence — this milestone's
// Validation section for OLT names only Name and OLTModelID as required,
// consistent with catalog.ProductCatalog's own optional Description.
//
// OLTModelID's requiredness only confirms the field was supplied, not
// that it resolves to a real OLTModel — a database-level concern the FK,
// not this method, enforces.
//
// ManagementIPAddress is deliberately not checked for being a
// well-formed IP address. This milestone's scope is recording that an
// OLT exists and where it can eventually be managed, not enforcing
// network configuration correctness — that judgment call is left to a
// future milestone that actually needs it (e.g. once something in this
// codebase parses or connects to the address), rather than adding
// validation nothing yet exercises.
func (o OLT) Validate() error {
	errs := validate.New()

	if !validate.Required(o.Name) {
		errs.Add("name", "is required")
	}
	if o.OLTModelID == uuid.Nil {
		errs.Add("olt_model_id", "is required")
	}

	return errs.Err()
}
