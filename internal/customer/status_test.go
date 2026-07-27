package customer_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
)

func TestCustomerStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []customer.CustomerStatus{
		customer.CustomerStatusActive,
		customer.CustomerStatusInactive,
		customer.CustomerStatusArchived,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestCustomerStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []customer.CustomerStatus{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"ARCHIVED", // wrong case
		"Deleted",  // not a defined status at all
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestCustomerStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := customer.CustomerStatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
