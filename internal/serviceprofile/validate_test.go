package serviceprofile_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
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

func validServiceProfile() serviceprofile.ServiceProfile {
	return serviceprofile.ServiceProfile{
		Name:   "Residential Internet",
		Status: serviceprofile.StatusActive,
	}
}

func TestServiceProfileValidate(t *testing.T) {
	if err := validServiceProfile().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, serviceprofile.ServiceProfile{}.Validate())
}

func TestServiceProfileValidateRequiresName(t *testing.T) {
	p := validServiceProfile()
	p.Name = ""

	assertInvalid(t, p.Validate())
}

func TestServiceProfileValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validServiceProfile()
	unrecognized.Status = serviceprofile.Status("Archived")
	assertInvalid(t, unrecognized.Validate())

	unset := validServiceProfile()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []serviceprofile.Status{
		serviceprofile.StatusActive,
		serviceprofile.StatusInactive,
	} {
		p := validServiceProfile()
		p.Status = s
		if err := p.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestServiceProfileValidateDescriptionIsOptional(t *testing.T) {
	p := validServiceProfile() // no description set
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	p.Description = "Standard residential internet service"
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
