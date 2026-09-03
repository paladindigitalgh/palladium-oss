package contact

import "strings"

// ContactRole classifies who a Contact is for the Customer, independent
// of ContactStatus (which tracks whether they are currently current). It
// is a distinct type, not a raw string, following the exact pattern of
// location.LocationType, inventory.DeviceStatus, and auth.Role: an
// unrecognized value is caught by validation instead of silently
// persisted.
type ContactRole string

// The five defined contact roles. There is no zero-value/default role —
// an empty ContactRole is invalid — so Role is effectively required on
// every Contact (see Contact.Validate in validate.go), for the same
// reason location.LocationType has no default type.
const (
	ContactRolePrimary   ContactRole = "Primary"
	ContactRoleBilling   ContactRole = "Billing"
	ContactRoleTechnical ContactRole = "Technical"
	ContactRoleEmergency ContactRole = "Emergency"
	ContactRoleOther     ContactRole = "Other"
)

// contactRoleOrder is the authoritative, ordered set of valid roles. It
// backs both Valid and validation error messages so the two can never
// disagree.
var contactRoleOrder = []ContactRole{
	ContactRolePrimary,
	ContactRoleBilling,
	ContactRoleTechnical,
	ContactRoleEmergency,
	ContactRoleOther,
}

// Valid reports whether r is one of the five defined ContactRole values.
func (r ContactRole) Valid() bool {
	for _, v := range contactRoleOrder {
		if r == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (r ContactRole) String() string {
	return string(r)
}

// contactRoleNames renders the defined roles as a comma-separated list,
// for use in validation error messages.
func contactRoleNames() string {
	names := make([]string, len(contactRoleOrder))
	for i, r := range contactRoleOrder {
		names[i] = string(r)
	}
	return strings.Join(names, ", ")
}
