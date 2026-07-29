package authentication_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
)

func TestAuthenticationTypeValidAcceptsDefinedValues(t *testing.T) {
	defined := []authentication.AuthenticationType{
		authentication.AuthenticationTypePassword,
		authentication.AuthenticationTypeSSHKey,
	}

	for _, typ := range defined {
		if !typ.Valid() {
			t.Errorf("%q.Valid() = false, want true", typ)
		}
	}
}

func TestAuthenticationTypeValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []authentication.AuthenticationType{
		"",         // zero value: there is no default type
		"password", // wrong case
		"Key",      // not a defined type
	}

	for _, typ := range cases {
		if typ.Valid() {
			t.Errorf("%q.Valid() = true, want false", typ)
		}
	}
}

func TestAuthenticationTypeStringReturnsUnderlyingValue(t *testing.T) {
	if got := authentication.AuthenticationTypeSSHKey.String(); got != "SSHKey" {
		t.Errorf("String() = %q, want %q", got, "SSHKey")
	}
}
