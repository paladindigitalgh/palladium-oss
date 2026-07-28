package accessnetwork_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
)

func TestAccessNetworkStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []accessnetwork.AccessNetworkStatus{
		accessnetwork.AccessNetworkStatusActive,
		accessnetwork.AccessNetworkStatusInactive,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestAccessNetworkStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []accessnetwork.AccessNetworkStatus{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"INACTIVE", // wrong case
		"Archived", // not a defined status for AccessNetwork
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestAccessNetworkStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := accessnetwork.AccessNetworkStatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
