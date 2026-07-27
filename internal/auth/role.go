package auth

import "strings"

// Role is a User's single authorization role. It is a distinct type, not
// a raw string threaded through the codebase, for the same reason
// inventory.DeviceStatus is: an unrecognized value is caught by
// validation instead of silently persisted.
//
// This is deliberately a single flat enum, not a hierarchy or a set of
// composable permissions: this milestone (RBAC v1) is intentionally
// simple — see CLAUDE.md's General Rule and this milestone's explicit
// "do not build a generic permission system" — and internal/authz is
// where a Role's actual capabilities are decided, not here. This package
// only knows that a Role is one of three fixed values; it says nothing
// about what any of them are allowed to do.
type Role string

// The three built-in roles. There is no "unknown"/zero-value role — an
// empty Role is invalid — so Role is effectively required on every User
// (see User.Validate in validate.go), for the same reason
// inventory.DeviceStatus has no default status.
const (
	RoleAdministrator Role = "Administrator"
	RoleOperator      Role = "Operator"
	RoleViewer        Role = "Viewer"
)

// roleOrder is the authoritative, ordered set of valid roles. It backs
// both Valid and validation error messages so the two can never disagree.
var roleOrder = []Role{
	RoleAdministrator,
	RoleOperator,
	RoleViewer,
}

// Valid reports whether r is one of the three defined roles.
func (r Role) Valid() bool {
	for _, v := range roleOrder {
		if r == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (r Role) String() string {
	return string(r)
}

// roleNames renders the defined roles as a comma-separated list, for use
// in validation error messages.
func roleNames() string {
	names := make([]string, len(roleOrder))
	for i, r := range roleOrder {
		names[i] = string(r)
	}
	return strings.Join(names, ", ")
}
