package contact

import "strings"

// ContactStatus is a Contact's lifecycle state, independent of
// ContactRole (which classifies who they are for the Customer). It is a
// distinct type, not a raw string, following the exact pattern of
// location.LocationStatus, inventory.DeviceStatus, and
// customer.CustomerStatus.
//
// This is a flat, two-value status, the same shape as
// location.LocationStatus and for the same reason: a Contact answers
// exactly one question here — is it currently valid to reach them about
// this account — not a richer lifecycle.
type ContactStatus string

// The two defined statuses. There is no zero-value/default status — an
// empty ContactStatus is invalid — so Status is effectively required on
// every Contact (see Contact.Validate in validate.go), for the same
// reason location.LocationStatus has no default status.
const (
	ContactStatusActive   ContactStatus = "Active"
	ContactStatusInactive ContactStatus = "Inactive"
)

// contactStatusOrder is the authoritative, ordered set of valid statuses.
// It backs both Valid and validation error messages so the two can never
// disagree.
var contactStatusOrder = []ContactStatus{
	ContactStatusActive,
	ContactStatusInactive,
}

// Valid reports whether s is one of the two defined ContactStatus
// values.
func (s ContactStatus) Valid() bool {
	for _, v := range contactStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s ContactStatus) String() string {
	return string(s)
}

// contactStatusNames renders the defined statuses as a comma-separated
// list, for use in validation error messages.
func contactStatusNames() string {
	names := make([]string, len(contactStatusOrder))
	for i, s := range contactStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
