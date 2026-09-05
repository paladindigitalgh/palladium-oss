package provider

import "strings"

// Status is a Provider's lifecycle state. It is a distinct type, not a
// raw string, following the exact pattern of catalog.CatalogStatus.
//
// Like CatalogStatus and serviceprofile.Status, this is a flat,
// two-value, reversible lifecycle: a wholesale partner pausing
// operations is expected to reactivate later, unlike product.ProductStatus's
// one-way Retired.
type Status string

// The two defined statuses. There is no zero-value/default status — an
// empty Status is invalid — so Status is effectively required on every
// Provider (see Provider.Validate in validate.go).
const (
	StatusActive   Status = "Active"
	StatusInactive Status = "Inactive"
)

// statusOrder is the authoritative, ordered set of valid statuses. It
// backs both Valid and validation error messages so the two can never
// disagree.
var statusOrder = []Status{
	StatusActive,
	StatusInactive,
}

// Valid reports whether s is one of the two defined Status values.
func (s Status) Valid() bool {
	for _, v := range statusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s Status) String() string {
	return string(s)
}

// statusNames renders the defined statuses as a comma-separated list,
// for use in validation error messages.
func statusNames() string {
	names := make([]string, len(statusOrder))
	for i, s := range statusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
