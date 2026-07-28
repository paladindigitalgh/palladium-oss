package accessnetwork

import "strings"

// AccessNetworkStatus is an AccessNetwork's lifecycle state. It is a
// distinct type, not a raw string, following the exact pattern of
// catalog.CatalogStatus.
//
// Like catalog.CatalogStatus, this is a flat, two-value lifecycle: an
// access network answers exactly one question here, "is it currently in
// service," not a richer provisioning-style lifecycle — that question
// does not apply at this level, only (eventually) to the individual OLTs
// and PON ports within it.
type AccessNetworkStatus string

// The two defined statuses. There is no zero-value/default status — an
// empty AccessNetworkStatus is invalid — so AccessNetworkStatus is
// effectively required on every AccessNetwork (see
// AccessNetwork.Validate in validate.go).
const (
	AccessNetworkStatusActive   AccessNetworkStatus = "Active"
	AccessNetworkStatusInactive AccessNetworkStatus = "Inactive"
)

// accessNetworkStatusOrder is the authoritative, ordered set of valid
// statuses. It backs both Valid and validation error messages so the two
// can never disagree.
var accessNetworkStatusOrder = []AccessNetworkStatus{
	AccessNetworkStatusActive,
	AccessNetworkStatusInactive,
}

// Valid reports whether s is one of the two defined AccessNetworkStatus
// values.
func (s AccessNetworkStatus) Valid() bool {
	for _, v := range accessNetworkStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s AccessNetworkStatus) String() string {
	return string(s)
}

// accessNetworkStatusNames renders the defined statuses as a
// comma-separated list, for use in validation error messages.
func accessNetworkStatusNames() string {
	names := make([]string, len(accessNetworkStatusOrder))
	for i, s := range accessNetworkStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
