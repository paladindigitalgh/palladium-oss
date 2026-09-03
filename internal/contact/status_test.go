package contact_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
)

func TestContactStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []contact.ContactStatus{
		contact.ContactStatusActive,
		contact.ContactStatusInactive,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestContactStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []contact.ContactStatus{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"INACTIVE", // wrong case
		"Archived", // not a defined status for Contact
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestContactStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := contact.ContactStatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
