package product_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

func TestProductStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []product.ProductStatus{
		product.ProductStatusActive,
		product.ProductStatusRetired,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestProductStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []product.ProductStatus{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"RETIRED",  // wrong case
		"Inactive", // not a defined status for Product
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestProductStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := product.ProductStatusRetired.String(); got != "Retired" {
		t.Errorf("String() = %q, want %q", got, "Retired")
	}
}
