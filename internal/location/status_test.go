package location_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
)

func TestLocationStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []location.LocationStatus{
		location.LocationStatusActive,
		location.LocationStatusInactive,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestLocationStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []location.LocationStatus{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"INACTIVE", // wrong case
		"Archived", // not a defined status for Location
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestLocationStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := location.LocationStatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
