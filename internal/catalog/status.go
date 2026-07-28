package catalog

import "strings"

// CatalogStatus is a ProductCatalog's lifecycle state. It is a distinct
// type, not a raw string, following the exact pattern of
// location.LocationStatus and customer.CustomerStatus.
//
// Like location.LocationStatus, this is a flat, two-value lifecycle: a
// catalog answers exactly one question here, "is it currently offered,"
// not a richer provisioning-style lifecycle — that question does not
// apply to a catalog at all, only (eventually) to the Products within it.
type CatalogStatus string

// The two defined statuses. There is no zero-value/default status — an
// empty CatalogStatus is invalid — so CatalogStatus is effectively
// required on every ProductCatalog (see ProductCatalog.Validate in
// validate.go).
const (
	CatalogStatusActive   CatalogStatus = "Active"
	CatalogStatusInactive CatalogStatus = "Inactive"
)

// catalogStatusOrder is the authoritative, ordered set of valid statuses.
// It backs both Valid and validation error messages so the two can never
// disagree.
var catalogStatusOrder = []CatalogStatus{
	CatalogStatusActive,
	CatalogStatusInactive,
}

// Valid reports whether s is one of the two defined CatalogStatus values.
func (s CatalogStatus) Valid() bool {
	for _, v := range catalogStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s CatalogStatus) String() string {
	return string(s)
}

// catalogStatusNames renders the defined statuses as a comma-separated
// list, for use in validation error messages.
func catalogStatusNames() string {
	names := make([]string, len(catalogStatusOrder))
	for i, s := range catalogStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
