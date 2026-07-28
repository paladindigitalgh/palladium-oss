package product

import "strings"

// ProductStatus is a Product's lifecycle state, independent of
// ProductCategory (which classifies what kind of offering it is). It is a
// distinct type, not a raw string, following the exact pattern of
// catalog.CatalogStatus.
//
// This is Retired, not Inactive, unlike catalog.CatalogStatus and
// location.LocationStatus: a Product's lifecycle answers "can this still
// be sold," and once a Product is retired that is understood to be a
// one-way transition — a discontinued offering is not expected to be
// silently re-activated later the way a temporarily-unused Location might
// be. Naming the value for what it means keeps that distinction visible
// in every response and validation error, rather than hiding it behind a
// generically-named "Inactive" that would read as reversible.
type ProductStatus string

// The two defined statuses. There is no zero-value/default status — an
// empty ProductStatus is invalid — so ProductStatus is effectively
// required on every Product (see Product.Validate in validate.go).
const (
	ProductStatusActive  ProductStatus = "Active"
	ProductStatusRetired ProductStatus = "Retired"
)

// productStatusOrder is the authoritative, ordered set of valid statuses.
// It backs both Valid and validation error messages so the two can never
// disagree.
var productStatusOrder = []ProductStatus{
	ProductStatusActive,
	ProductStatusRetired,
}

// Valid reports whether s is one of the two defined ProductStatus values.
func (s ProductStatus) Valid() bool {
	for _, v := range productStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s ProductStatus) String() string {
	return string(s)
}

// productStatusNames renders the defined statuses as a comma-separated
// list, for use in validation error messages.
func productStatusNames() string {
	names := make([]string, len(productStatusOrder))
	for i, s := range productStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
