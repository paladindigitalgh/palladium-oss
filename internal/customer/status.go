package customer

import "strings"

// CustomerStatus is a Customer's lifecycle state, independent of
// CustomerType (which classifies who they are). It is a distinct type,
// not a raw string, following the exact pattern of inventory.DeviceStatus
// and auth.Role.
//
// This is deliberately a flat, three-value lifecycle, not the richer
// inventory lifecycle (Ordered -> Received -> ... -> Disposed, see
// inventory.DeviceStatus) — a Customer's status here answers one
// question, "should this identity currently be treated as active," not
// "where are they in a provisioning workflow." That richer question
// belongs to Services once that phase exists.
type CustomerStatus string

// The three defined statuses. There is no zero-value/default status — an
// empty CustomerStatus is invalid — so CustomerStatus is effectively
// required on every Customer (see Customer.Validate in validate.go), for
// the same reason inventory.DeviceStatus has no default status.
const (
	CustomerStatusActive   CustomerStatus = "Active"
	CustomerStatusInactive CustomerStatus = "Inactive"
	CustomerStatusArchived CustomerStatus = "Archived"
)

// customerStatusOrder is the authoritative, ordered set of valid statuses.
// It backs both Valid and validation error messages so the two can never
// disagree.
var customerStatusOrder = []CustomerStatus{
	CustomerStatusActive,
	CustomerStatusInactive,
	CustomerStatusArchived,
}

// Valid reports whether s is one of the three defined CustomerStatus
// values.
func (s CustomerStatus) Valid() bool {
	for _, v := range customerStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s CustomerStatus) String() string {
	return string(s)
}

// customerStatusNames renders the defined statuses as a comma-separated
// list, for use in validation error messages.
func customerStatusNames() string {
	names := make([]string, len(customerStatusOrder))
	for i, s := range customerStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
