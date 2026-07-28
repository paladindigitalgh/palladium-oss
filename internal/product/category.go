package product

import "strings"

// ProductCategory classifies what kind of offering a Product is. It is a
// distinct type, not a raw string, following the exact pattern of
// location.LocationType.
type ProductCategory string

// The six defined categories. There is no zero-value/default category —
// an empty ProductCategory is invalid — so ProductCategory is effectively
// required on every Product (see Product.Validate in validate.go).
const (
	ProductCategoryInternet    ProductCategory = "Internet"
	ProductCategoryVoice       ProductCategory = "Voice"
	ProductCategoryIPTV        ProductCategory = "IPTV"
	ProductCategoryTransport   ProductCategory = "Transport"
	ProductCategoryManagedWiFi ProductCategory = "ManagedWiFi"
	ProductCategoryOther       ProductCategory = "Other"
)

// productCategoryOrder is the authoritative, ordered set of valid
// categories. It backs both Valid and validation error messages so the
// two can never disagree.
var productCategoryOrder = []ProductCategory{
	ProductCategoryInternet,
	ProductCategoryVoice,
	ProductCategoryIPTV,
	ProductCategoryTransport,
	ProductCategoryManagedWiFi,
	ProductCategoryOther,
}

// Valid reports whether c is one of the six defined ProductCategory
// values.
func (c ProductCategory) Valid() bool {
	for _, v := range productCategoryOrder {
		if c == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (c ProductCategory) String() string {
	return string(c)
}

// productCategoryNames renders the defined categories as a
// comma-separated list, for use in validation error messages.
func productCategoryNames() string {
	names := make([]string, len(productCategoryOrder))
	for i, c := range productCategoryOrder {
		names[i] = string(c)
	}
	return strings.Join(names, ", ")
}
