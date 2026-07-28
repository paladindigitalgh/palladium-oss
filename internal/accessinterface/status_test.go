package accessinterface_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
)

func TestStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []accessinterface.Status{
		accessinterface.StatusActive,
		accessinterface.StatusDisabled,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []accessinterface.Status{
		"",          // zero value: there is no default status
		"active",    // wrong case
		"Suspended", // not a defined status
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := accessinterface.StatusActive.String(); got != "Active" {
		t.Errorf("String() = %q, want %q", got, "Active")
	}
}
