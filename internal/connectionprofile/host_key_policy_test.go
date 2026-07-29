package connectionprofile_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
)

func TestHostKeyPolicyValidAcceptsDefinedValues(t *testing.T) {
	defined := []connectionprofile.HostKeyPolicy{
		connectionprofile.HostKeyPolicyStrict,
		connectionprofile.HostKeyPolicyInsecure,
	}

	for _, p := range defined {
		if !p.Valid() {
			t.Errorf("%q.Valid() = false, want true", p)
		}
	}
}

func TestHostKeyPolicyValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []connectionprofile.HostKeyPolicy{
		"",         // zero value: there is no default policy
		"strict",   // wrong case
		"Disabled", // not a defined policy
	}

	for _, p := range cases {
		if p.Valid() {
			t.Errorf("%q.Valid() = true, want false", p)
		}
	}
}

func TestHostKeyPolicyStringReturnsUnderlyingValue(t *testing.T) {
	if got := connectionprofile.HostKeyPolicyInsecure.String(); got != "Insecure" {
		t.Errorf("String() = %q, want %q", got, "Insecure")
	}
}
