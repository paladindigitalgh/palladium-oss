package location

import "strings"

// LocationStatus is a Location's lifecycle state, independent of
// LocationType (which classifies what it is used for). It is a distinct
// type, not a raw string, following the exact pattern of
// inventory.DeviceStatus, auth.Role, and customer.CustomerStatus.
//
// This is a flat, two-value status — simpler even than
// customer.CustomerStatus's three — because a Location answers exactly
// one question here: is it currently in use. There is no "Archived"
// state of its own; a Location's own lifecycle is intentionally this
// minimal in v1 (see the package doc comment on what is deliberately not
// modeled yet).
type LocationStatus string

// The two defined statuses. There is no zero-value/default status — an
// empty LocationStatus is invalid — so Status is effectively required on
// every Location (see Location.Validate in validate.go), for the same
// reason inventory.DeviceStatus has no default status.
const (
	LocationStatusActive   LocationStatus = "Active"
	LocationStatusInactive LocationStatus = "Inactive"
)

// locationStatusOrder is the authoritative, ordered set of valid
// statuses. It backs both Valid and validation error messages so the two
// can never disagree.
var locationStatusOrder = []LocationStatus{
	LocationStatusActive,
	LocationStatusInactive,
}

// Valid reports whether s is one of the two defined LocationStatus
// values.
func (s LocationStatus) Valid() bool {
	for _, v := range locationStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s LocationStatus) String() string {
	return string(s)
}

// locationStatusNames renders the defined statuses as a comma-separated
// list, for use in validation error messages.
func locationStatusNames() string {
	names := make([]string, len(locationStatusOrder))
	for i, s := range locationStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
