package olt_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/accessnetwork/validate_test.go's helper
// of the same name: every domain package's Validate() must return an
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

func validOLT() olt.OLT {
	return olt.OLT{
		AccessNetworkID: uuid.New(),
		Name:            "OLT-01",
		OLTModelID:      uuid.New(),
	}
}

func TestOLTValidate(t *testing.T) {
	if err := validOLT().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, olt.OLT{}.Validate())
}

func TestOLTValidateRequiresAccessNetworkID(t *testing.T) {
	o := validOLT()
	o.AccessNetworkID = uuid.Nil

	assertInvalid(t, o.Validate())
}

func TestOLTValidateRequiresName(t *testing.T) {
	o := validOLT()
	o.Name = ""

	assertInvalid(t, o.Validate())
}

func TestOLTValidateRequiresOLTModelID(t *testing.T) {
	o := validOLT()
	o.OLTModelID = uuid.Nil

	assertInvalid(t, o.Validate())
}

func TestOLTValidateManagementIPAddressAndDescriptionAreOptional(t *testing.T) {
	o := validOLT() // neither set
	if err := o.Validate(); err != nil {
		t.Errorf("Validate() (no optional fields) = %v, want nil", err)
	}

	o.ManagementIPAddress = "10.0.0.1"
	o.Description = "Primary OLT for the north region"
	if err := o.Validate(); err != nil {
		t.Errorf("Validate() (with optional fields) = %v, want nil", err)
	}
}
