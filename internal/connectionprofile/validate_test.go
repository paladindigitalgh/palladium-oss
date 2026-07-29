package connectionprofile_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
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

func validConnectionProfile() connectionprofile.ConnectionProfile {
	return connectionprofile.ConnectionProfile{
		Name:          "Standard SSH",
		HostKeyPolicy: connectionprofile.HostKeyPolicyStrict,
	}
}

func TestConnectionProfileValidate(t *testing.T) {
	if err := validConnectionProfile().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, connectionprofile.ConnectionProfile{}.Validate())
}

func TestConnectionProfileValidateRequiresName(t *testing.T) {
	p := validConnectionProfile()
	p.Name = ""

	assertInvalid(t, p.Validate())
}

func TestConnectionProfileValidateRequiresKnownHostKeyPolicy(t *testing.T) {
	unrecognized := validConnectionProfile()
	unrecognized.HostKeyPolicy = connectionprofile.HostKeyPolicy("Disabled")
	assertInvalid(t, unrecognized.Validate())

	unset := validConnectionProfile()
	unset.HostKeyPolicy = ""
	assertInvalid(t, unset.Validate())

	for _, p := range []connectionprofile.HostKeyPolicy{
		connectionprofile.HostKeyPolicyStrict,
		connectionprofile.HostKeyPolicyInsecure,
	} {
		cp := validConnectionProfile()
		cp.HostKeyPolicy = p
		if err := cp.Validate(); err != nil {
			t.Errorf("Validate() (host key policy %q) = %v, want nil", p, err)
		}
	}
}

// TestConnectionProfileValidateDoesNotRequireProtocolPortAuthenticationOrTimeout
// proves this milestone's explicit "only Name unique" reading: a
// ConnectionProfile with none of Protocol, Port, AuthenticationID, or
// Timeout set is still valid — see validate.go's own doc comment for
// the full reasoning.
func TestConnectionProfileValidateDoesNotRequireProtocolPortAuthenticationOrTimeout(t *testing.T) {
	p := validConnectionProfile() // Protocol, Port, AuthenticationID, Timeout, Description all zero-valued
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for a profile with only Name and HostKeyPolicy set", err)
	}
}

func TestConnectionProfileValidateAcceptsFullySpecifiedProfile(t *testing.T) {
	authID := uuid.New()
	p := connectionprofile.ConnectionProfile{
		Name:             "Fully Specified SSH",
		Protocol:         "SSH",
		Port:             22,
		AuthenticationID: &authID,
		Timeout:          30 * time.Second,
		HostKeyPolicy:    connectionprofile.HostKeyPolicyStrict,
		Description:      "Standard SSH profile for lab OLTs",
	}
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}
