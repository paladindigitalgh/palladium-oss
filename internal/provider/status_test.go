package provider_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/provider"
)

func TestStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []provider.Status{
		provider.StatusActive,
		provider.StatusInactive,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []provider.Status{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"INACTIVE", // wrong case
		"Archived", // not a defined status for Provider
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := provider.StatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
