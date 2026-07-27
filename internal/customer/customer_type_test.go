package customer_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
)

func TestCustomerTypeValidAcceptsDefinedValues(t *testing.T) {
	defined := []customer.CustomerType{
		customer.CustomerTypeResidential,
		customer.CustomerTypeBusiness,
		customer.CustomerTypeGovernment,
		customer.CustomerTypeInternal,
	}

	for _, ct := range defined {
		if !ct.Valid() {
			t.Errorf("%q.Valid() = false, want true", ct)
		}
	}
}

func TestCustomerTypeValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []customer.CustomerType{
		"",            // zero value: there is no default type
		"residential", // wrong case
		"BUSINESS",    // wrong case
		"Nonprofit",   // not a defined type at all
	}

	for _, ct := range cases {
		if ct.Valid() {
			t.Errorf("%q.Valid() = true, want false", ct)
		}
	}
}

func TestCustomerTypeStringReturnsUnderlyingValue(t *testing.T) {
	if got := customer.CustomerTypeBusiness.String(); got != "Business" {
		t.Errorf("String() = %q, want %q", got, "Business")
	}
}
