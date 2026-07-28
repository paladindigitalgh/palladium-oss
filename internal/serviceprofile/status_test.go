package serviceprofile_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
)

func TestStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []serviceprofile.Status{
		serviceprofile.StatusActive,
		serviceprofile.StatusInactive,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []serviceprofile.Status{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"INACTIVE", // wrong case
		"Archived", // not a defined status for ServiceProfile
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := serviceprofile.StatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
