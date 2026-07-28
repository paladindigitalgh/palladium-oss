package accessnetwork_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
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

func validAccessNetwork() accessnetwork.AccessNetwork {
	return accessnetwork.AccessNetwork{
		Name:   "North Region GPON",
		Status: accessnetwork.AccessNetworkStatusActive,
	}
}

func TestAccessNetworkValidate(t *testing.T) {
	if err := validAccessNetwork().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, accessnetwork.AccessNetwork{}.Validate())
}

func TestAccessNetworkValidateRequiresName(t *testing.T) {
	a := validAccessNetwork()
	a.Name = ""

	assertInvalid(t, a.Validate())
}

func TestAccessNetworkValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validAccessNetwork()
	unrecognized.Status = accessnetwork.AccessNetworkStatus("Archived")
	assertInvalid(t, unrecognized.Validate())

	unset := validAccessNetwork()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []accessnetwork.AccessNetworkStatus{
		accessnetwork.AccessNetworkStatusActive,
		accessnetwork.AccessNetworkStatusInactive,
	} {
		a := validAccessNetwork()
		a.Status = s
		if err := a.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestAccessNetworkValidateDescriptionIsOptional(t *testing.T) {
	a := validAccessNetwork() // no description set
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	a.Description = "Covers the northern service area"
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
