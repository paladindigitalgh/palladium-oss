package customer

import "strings"

// CustomerType classifies who a Customer is, independent of CustomerStatus
// (which tracks their lifecycle state). It is a distinct type, not a raw
// string, following the exact pattern of inventory.DeviceStatus and
// auth.Role: an unrecognized value is caught by validation instead of
// silently persisted.
//
// Internal exists alongside the three real-world categories for the same
// reason ISPs actually need it in practice: staff test lines, lab
// circuits, and other accounts that are not a paying resident, business,
// or government entity still need to be modeled as Customers so Services
// (a later phase) have someone to attach to — not exempted from the
// domain model as a special case.
type CustomerType string

// The four defined customer types. There is no zero-value/default type —
// an empty CustomerType is invalid — so CustomerType is effectively
// required on every Customer (see Customer.Validate in validate.go), for
// the same reason inventory.DeviceStatus has no default status.
const (
	CustomerTypeResidential CustomerType = "Residential"
	CustomerTypeBusiness    CustomerType = "Business"
	CustomerTypeGovernment  CustomerType = "Government"
	CustomerTypeInternal    CustomerType = "Internal"
)

// customerTypeOrder is the authoritative, ordered set of valid types. It
// backs both Valid and validation error messages so the two can never
// disagree.
var customerTypeOrder = []CustomerType{
	CustomerTypeResidential,
	CustomerTypeBusiness,
	CustomerTypeGovernment,
	CustomerTypeInternal,
}

// Valid reports whether c is one of the four defined CustomerType values.
func (c CustomerType) Valid() bool {
	for _, v := range customerTypeOrder {
		if c == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (c CustomerType) String() string {
	return string(c)
}

// customerTypeNames renders the defined types as a comma-separated list,
// for use in validation error messages.
func customerTypeNames() string {
	names := make([]string, len(customerTypeOrder))
	for i, c := range customerTypeOrder {
		names[i] = string(c)
	}
	return strings.Join(names, ", ")
}
