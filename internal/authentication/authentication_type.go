package authentication

import "strings"

// AuthenticationType identifies which of Authentication's two credential
// fields (Password or PrivateKey) is actually in use. It is a distinct
// type, not a raw string, following the exact pattern of
// accessinterface.Technology.
//
// Unlike a status field elsewhere in this codebase, AuthenticationType
// does not describe a lifecycle — it governs which of two other fields
// Validate requires (see validate.go): "Password required for Password
// auth, PrivateKey required for SSHKey auth," per this milestone's exact
// wording.
type AuthenticationType string

// The two defined authentication types. There is no zero-value/default
// type — an empty AuthenticationType is invalid — so AuthenticationType
// is effectively required on every Authentication (see
// Authentication.Validate in validate.go).
const (
	AuthenticationTypePassword AuthenticationType = "Password"
	AuthenticationTypeSSHKey   AuthenticationType = "SSHKey"
)

// authenticationTypeOrder is the authoritative, ordered set of valid
// types. It backs both Valid and validation error messages so the two
// can never disagree.
var authenticationTypeOrder = []AuthenticationType{
	AuthenticationTypePassword,
	AuthenticationTypeSSHKey,
}

// Valid reports whether t is one of the two defined AuthenticationType
// values.
func (t AuthenticationType) Valid() bool {
	for _, defined := range authenticationTypeOrder {
		if t == defined {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (t AuthenticationType) String() string {
	return string(t)
}

// authenticationTypeNames renders the defined types as a comma-separated
// list, for use in validation error messages.
func authenticationTypeNames() string {
	names := make([]string, len(authenticationTypeOrder))
	for i, t := range authenticationTypeOrder {
		names[i] = string(t)
	}
	return strings.Join(names, ", ")
}
