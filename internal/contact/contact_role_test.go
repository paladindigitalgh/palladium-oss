package contact_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
)

func TestContactRoleValidAcceptsDefinedValues(t *testing.T) {
	defined := []contact.ContactRole{
		contact.ContactRolePrimary,
		contact.ContactRoleBilling,
		contact.ContactRoleTechnical,
		contact.ContactRoleEmergency,
		contact.ContactRoleOther,
	}

	for _, r := range defined {
		if !r.Valid() {
			t.Errorf("%q.Valid() = false, want true", r)
		}
	}
}

func TestContactRoleValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []contact.ContactRole{
		"",          // zero value: there is no default role
		"primary",   // wrong case
		"BILLING",   // wrong case
		"Executive", // not a defined role at all
	}

	for _, r := range cases {
		if r.Valid() {
			t.Errorf("%q.Valid() = true, want false", r)
		}
	}
}

func TestContactRoleStringReturnsUnderlyingValue(t *testing.T) {
	if got := contact.ContactRoleTechnical.String(); got != "Technical" {
		t.Errorf("String() = %q, want %q", got, "Technical")
	}
}
