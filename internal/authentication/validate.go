package authentication

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether a has every required field set: a present
// Name, an AuthenticationType that is one of its defined values (see
// authentication_type.go), a present Username, and — depending on
// AuthenticationType — a present Password or PrivateKey, per this
// milestone's exact rules:
//
//   - Password is required when AuthenticationType is
//     AuthenticationTypePassword.
//   - PrivateKey is required when AuthenticationType is
//     AuthenticationTypeSSHKey.
//
// Whichever of Password/PrivateKey is not required is never checked —
// an Authentication of type SSHKey with an empty Password is exactly
// what every valid SSHKey-type record looks like, not a partially-filled
// one. Name's uniqueness (this milestone's other explicit rule) is not
// checked here: Validate only ever answers "is this record well-formed
// in isolation," the same reasoning every other domain's Validate in
// this codebase documents for why a cross-row check like uniqueness
// belongs to the database (a UNIQUE constraint) and the repository layer
// that translates its violation into apperror.KindConflict, not to this
// function.
func (a Authentication) Validate() error {
	errs := validate.New()

	if !validate.Required(a.Name) {
		errs.Add("name", "is required")
	}
	if !a.AuthenticationType.Valid() {
		errs.Add("authentication_type", fmt.Sprintf("must be one of: %s", authenticationTypeNames()))
	}
	if !validate.Required(a.Username) {
		errs.Add("username", "is required")
	}
	if a.AuthenticationType == AuthenticationTypePassword && !validate.Required(a.Password) {
		errs.Add("password", "is required for Password authentication")
	}
	if a.AuthenticationType == AuthenticationTypeSSHKey && !validate.Required(a.PrivateKey) {
		errs.Add("private_key", "is required for SSHKey authentication")
	}

	return errs.Err()
}
