package auth_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
)

func TestUserStatusValidAcceptsDefinedValues(t *testing.T) {
	for _, status := range []auth.UserStatus{auth.UserStatusActive, auth.UserStatusInactive} {
		if !status.Valid() {
			t.Errorf("%q.Valid() = false, want true", status)
		}
	}
}

func TestUserStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []auth.UserStatus{
		"",          // zero value: there is no default status
		"active",    // wrong case
		"INACTIVE",  // wrong case
		"Suspended", // not a defined status at all
	}

	for _, status := range cases {
		if status.Valid() {
			t.Errorf("%q.Valid() = true, want false", status)
		}
	}
}

func TestUserStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := auth.UserStatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
