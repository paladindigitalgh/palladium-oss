package authentication_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/catalog/validate_test.go's helper of
// the same name: every domain package's Validate() must return an
// *apperror.Error of KindInvalid.
func assertInvalid(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("Validate() error is not an *apperror.Error: %v", err)
	}
	if appErr.Kind != apperror.KindInvalid {
		t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindInvalid)
	}
}

func validPasswordAuthentication() authentication.Authentication {
	return authentication.Authentication{
		Name:               "Default Device Login",
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "admin",
		Password:           "hunter2",
	}
}

func validSSHKeyAuthentication() authentication.Authentication {
	return authentication.Authentication{
		Name:               "Default Device SSH Key",
		AuthenticationType: authentication.AuthenticationTypeSSHKey,
		Username:           "admin",
		PrivateKey:         "-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----",
	}
}

func TestAuthenticationValidatePasswordType(t *testing.T) {
	if err := validPasswordAuthentication().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, authentication.Authentication{}.Validate())
}

func TestAuthenticationValidateSSHKeyType(t *testing.T) {
	if err := validSSHKeyAuthentication().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestAuthenticationValidateRequiresName(t *testing.T) {
	a := validPasswordAuthentication()
	a.Name = ""

	assertInvalid(t, a.Validate())
}

func TestAuthenticationValidateRequiresKnownAuthenticationType(t *testing.T) {
	unrecognized := validPasswordAuthentication()
	unrecognized.AuthenticationType = authentication.AuthenticationType("Token")
	assertInvalid(t, unrecognized.Validate())

	unset := validPasswordAuthentication()
	unset.AuthenticationType = ""
	assertInvalid(t, unset.Validate())
}

func TestAuthenticationValidateRequiresUsername(t *testing.T) {
	a := validPasswordAuthentication()
	a.Username = ""

	assertInvalid(t, a.Validate())
}

// TestAuthenticationValidateRequiresPasswordForPasswordType proves this
// milestone's exact rule: "Password required for Password auth."
func TestAuthenticationValidateRequiresPasswordForPasswordType(t *testing.T) {
	a := validPasswordAuthentication()
	a.Password = ""

	assertInvalid(t, a.Validate())
}

// TestAuthenticationValidateRequiresPrivateKeyForSSHKeyType proves this
// milestone's exact rule: "PrivateKey required for SSHKey auth."
func TestAuthenticationValidateRequiresPrivateKeyForSSHKeyType(t *testing.T) {
	a := validSSHKeyAuthentication()
	a.PrivateKey = ""

	assertInvalid(t, a.Validate())
}

// TestAuthenticationValidateDoesNotRequirePasswordForSSHKeyType proves
// an SSHKey-type record with no Password is valid — Password is simply
// not this type's credential field, not a partially-filled record.
func TestAuthenticationValidateDoesNotRequirePasswordForSSHKeyType(t *testing.T) {
	a := validSSHKeyAuthentication()
	a.Password = "" // already empty in the builder; explicit for clarity

	if err := a.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for an SSHKey-type record with no Password", err)
	}
}

// TestAuthenticationValidateDoesNotRequirePrivateKeyForPasswordType is
// TestAuthenticationValidateDoesNotRequirePasswordForSSHKeyType's
// counterpart for the other type.
func TestAuthenticationValidateDoesNotRequirePrivateKeyForPasswordType(t *testing.T) {
	a := validPasswordAuthentication()
	a.PrivateKey = ""

	if err := a.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for a Password-type record with no PrivateKey", err)
	}
}
