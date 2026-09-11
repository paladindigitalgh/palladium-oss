package product_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

func TestServiceTypeValidAcceptsDefinedValues(t *testing.T) {
	defined := []product.ServiceType{
		product.ServiceTypeResidential,
		product.ServiceTypeBusiness,
		product.ServiceTypeInternal,
	}

	for _, tt := range defined {
		if !tt.Valid() {
			t.Errorf("%q.Valid() = false, want true", tt)
		}
	}
}

func TestServiceTypeValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []product.ServiceType{
		"",            // zero value: there is no default service type
		"residential", // wrong case
		"BUSINESS",    // wrong case
		"Enterprise",  // not a defined service type at all
	}

	for _, tt := range cases {
		if tt.Valid() {
			t.Errorf("%q.Valid() = true, want false", tt)
		}
	}
}

func TestServiceTypeStringReturnsUnderlyingValue(t *testing.T) {
	if got := product.ServiceTypeBusiness.String(); got != "Business" {
		t.Errorf("String() = %q, want %q", got, "Business")
	}
}
