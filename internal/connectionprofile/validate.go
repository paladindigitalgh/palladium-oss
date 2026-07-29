package connectionprofile

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether p has every required field set: a present
// Name and a HostKeyPolicy that is one of its defined values (see
// host_key_policy.go).
//
// Protocol, Port, AuthenticationID, Timeout, and Description are all
// left unvalidated. This is a deliberate reading of this milestone's
// Rules section for ConnectionProfile, which names exactly one
// requirement — "Name unique" (the uniqueness half of which belongs to
// the database, not here; see the same reasoning
// authentication.Authentication.Validate's own doc comment gives) — in
// sharp contrast to Authentication's own Rules section in the same
// milestone, which names four. Protocol has no listed set of valid
// values in this milestone's spec at all (unlike HostKeyPolicy, which
// lists Strict/Insecure explicitly), so it is a plain string, not a
// closed enum, and is never required to be non-empty; Port,
// AuthenticationID, and Timeout are all meaningful in their zero-value
// state (a ConnectionProfile not yet bound to a specific port, a
// specific Authentication, or a specific timeout is a legitimate
// template — see model.go's own doc comment on AuthenticationID).
// HostKeyPolicy is the one exception: it is a closed enum this package
// itself defines (see host_key_policy.go), and every closed enum in
// this codebase is validated as required — an unvalidated enum field
// would accept values nothing downstream could ever interpret, which is
// not the same as "optional."
func (p ConnectionProfile) Validate() error {
	errs := validate.New()

	if !validate.Required(p.Name) {
		errs.Add("name", "is required")
	}
	if !p.HostKeyPolicy.Valid() {
		errs.Add("host_key_policy", fmt.Sprintf("must be one of: %s", hostKeyPolicyNames()))
	}

	return errs.Err()
}
