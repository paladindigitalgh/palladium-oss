package product

import "strings"

// ServiceType classifies which subscriber segment a Product is sold to.
// It is a distinct type, not a raw string, following the exact pattern
// of ProductCategory.
type ServiceType string

// The three defined service types. There is no zero-value/default
// service type — an empty ServiceType is invalid — so ServiceType is
// effectively required on every Product (see Product.Validate in
// validate.go).
const (
	ServiceTypeResidential ServiceType = "Residential"
	ServiceTypeBusiness    ServiceType = "Business"
	ServiceTypeInternal    ServiceType = "Internal"
)

// serviceTypeOrder is the authoritative, ordered set of valid service
// types. It backs both Valid and validation error messages so the two
// can never disagree.
var serviceTypeOrder = []ServiceType{
	ServiceTypeResidential,
	ServiceTypeBusiness,
	ServiceTypeInternal,
}

// Valid reports whether t is one of the three defined ServiceType
// values.
func (t ServiceType) Valid() bool {
	for _, v := range serviceTypeOrder {
		if t == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (t ServiceType) String() string {
	return string(t)
}

// serviceTypeNames renders the defined service types as a
// comma-separated list, for use in validation error messages.
func serviceTypeNames() string {
	names := make([]string, len(serviceTypeOrder))
	for i, t := range serviceTypeOrder {
		names[i] = string(t)
	}
	return strings.Join(names, ", ")
}
