package connectionprofile

import "strings"

// HostKeyPolicy governs whether a future connection built from this
// ConnectionProfile will verify the remote host's key. It is a distinct
// type, not a raw string, following the exact pattern of
// authentication.AuthenticationType.
//
// This maps conceptually to internal/platform/ssh.Config's own
// StrictHostKeyChecking field — but deliberately as a two-value enum
// with no zero-value default, not a bool. That is not a cosmetic choice:
// ssh.Config.StrictHostKeyChecking's own doc comment documents, at
// length, that a plain bool's zero value (false) silently means
// "insecure" unless every caller remembers to opt in explicitly.
// HostKeyPolicy has no such trap — an empty HostKeyPolicy is invalid
// (see Validate in validate.go, which requires one of the two values
// below), so a ConnectionProfile can never end up insecure merely
// because a caller forgot to set a field.
type HostKeyPolicy string

// The two defined policies. There is no zero-value/default policy — an
// empty HostKeyPolicy is invalid — so HostKeyPolicy is effectively
// required on every ConnectionProfile (see ConnectionProfile.Validate in
// validate.go).
const (
	// HostKeyPolicyStrict means a future connection must verify the
	// remote host's key (e.g. against a known_hosts file) before
	// proceeding.
	HostKeyPolicyStrict HostKeyPolicy = "Strict"

	// HostKeyPolicyInsecure means a future connection accepts the
	// remote host's key unconditionally — explicitly named "Insecure,"
	// per this milestone's own naming, so choosing it is never an
	// accident. Intended for labs, exactly as
	// internal/platform/ssh.Config's own StrictHostKeyChecking: false
	// escape hatch is documented for.
	HostKeyPolicyInsecure HostKeyPolicy = "Insecure"
)

// hostKeyPolicyOrder is the authoritative, ordered set of valid
// policies. It backs both Valid and validation error messages so the
// two can never disagree.
var hostKeyPolicyOrder = []HostKeyPolicy{
	HostKeyPolicyStrict,
	HostKeyPolicyInsecure,
}

// Valid reports whether p is one of the two defined HostKeyPolicy
// values.
func (p HostKeyPolicy) Valid() bool {
	for _, defined := range hostKeyPolicyOrder {
		if p == defined {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (p HostKeyPolicy) String() string {
	return string(p)
}

// hostKeyPolicyNames renders the defined policies as a comma-separated
// list, for use in validation error messages.
func hostKeyPolicyNames() string {
	names := make([]string, len(hostKeyPolicyOrder))
	for i, p := range hostKeyPolicyOrder {
		names[i] = string(p)
	}
	return strings.Join(names, ", ")
}
