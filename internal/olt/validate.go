package olt

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether o has every required field set: a present
// AccessNetworkID, a present Name, and a Vendor that is one of its
// defined values (see vendor.go). Model, ManagementIPAddress, and
// Description are all optional and are never checked for presence — this
// milestone's Validation section for OLT names only AccessNetworkID,
// Name, and Vendor as required, consistent with catalog.ProductCatalog's
// own optional Description.
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

	if o.AccessNetworkID == uuid.Nil {
		errs.Add("access_network_id", "is required")
	}
	if !validate.Required(o.Name) {
		errs.Add("name", "is required")
	}
	if !o.Vendor.Valid() {
		errs.Add("vendor", fmt.Sprintf("must be one of: %s", vendorNames()))
	}

	return errs.Err()
}
