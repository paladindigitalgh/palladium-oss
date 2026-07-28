package accessinterface_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/olt/validate_test.go's helper of the
// same name: every domain package's Validate() must return an
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

func validAccessInterface() accessinterface.AccessInterface {
	return accessinterface.AccessInterface{
		PONPortID:  uuid.New(),
		Technology: accessinterface.TechnologyGPON,
		Name:       "gpon-0/1/1",
		Status:     accessinterface.StatusActive,
	}
}

func TestAccessInterfaceValidate(t *testing.T) {
	if err := validAccessInterface().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, accessinterface.AccessInterface{}.Validate())
}

func TestAccessInterfaceValidateRequiresPONPortID(t *testing.T) {
	a := validAccessInterface()
	a.PONPortID = uuid.Nil

	assertInvalid(t, a.Validate())
}

func TestAccessInterfaceValidateRequiresKnownTechnology(t *testing.T) {
	unrecognized := validAccessInterface()
	unrecognized.Technology = accessinterface.Technology("EPON")
	assertInvalid(t, unrecognized.Validate())

	unset := validAccessInterface()
	unset.Technology = ""
	assertInvalid(t, unset.Validate())

	for _, tech := range []accessinterface.Technology{
		accessinterface.TechnologyGPON,
		accessinterface.TechnologyXGSPON,
		accessinterface.TechnologyActiveEthernet,
		accessinterface.TechnologyOther,
	} {
		a := validAccessInterface()
		a.Technology = tech
		if err := a.Validate(); err != nil {
			t.Errorf("Validate() (technology %q) = %v, want nil", tech, err)
		}
	}
}

func TestAccessInterfaceValidateRequiresName(t *testing.T) {
	a := validAccessInterface()
	a.Name = ""

	assertInvalid(t, a.Validate())
}

func TestAccessInterfaceValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validAccessInterface()
	unrecognized.Status = accessinterface.Status("Suspended")
	assertInvalid(t, unrecognized.Validate())

	unset := validAccessInterface()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []accessinterface.Status{
		accessinterface.StatusActive,
		accessinterface.StatusDisabled,
	} {
		a := validAccessInterface()
		a.Status = s
		if err := a.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestAccessInterfaceValidateDescriptionIsOptional(t *testing.T) {
	a := validAccessInterface()
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	a.Description = "Serves the north-side splitter cabinet"
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
