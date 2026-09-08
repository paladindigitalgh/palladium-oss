package oltmodel

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether m has every required field set: a present
// Name, a Vendor that is one of its defined values (see vendor.go), and
// a positive PONPortCount. Description is optional and is never checked
// for presence, consistent with provider.Provider.Validate.
//
// PONPortCount must be a positive integer: zero or negative would mean
// internal/olt/service.OLTService.Create's auto-port-creation cascade
// (see that package's doc comment) silently creates no ports, or panics
// trying to. This is the one place in this domain a genuinely enforced
// business rule about port count exists — everywhere downstream of here
// simply trusts a validated OLTModel's PONPortCount.
func (m OLTModel) Validate() error {
	errs := validate.New()

	if !validate.Required(m.Name) {
		errs.Add("name", "is required")
	}
	if !m.Vendor.Valid() {
		errs.Add("vendor", fmt.Sprintf("must be one of: %s", vendorNames()))
	}
	if m.PONPortCount <= 0 {
		errs.Add("pon_port_count", "must be a positive integer")
	}

	return errs.Err()
}
