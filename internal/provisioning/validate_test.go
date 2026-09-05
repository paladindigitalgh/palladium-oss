package provisioning_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// assertInvalid mirrors internal/product/validate_test.go's helper of the
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

func validProvisioningProfile() provisioning.ProvisioningProfile {
	return provisioning.ProvisioningProfile{
		ProductID:   uuid.New(),
		Vendor:      "Kontron",
		ProfileName: "RES-500M",
	}
}

func TestProvisioningProfileValidate(t *testing.T) {
	if err := validProvisioningProfile().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, provisioning.ProvisioningProfile{}.Validate())
}

func TestProvisioningProfileValidateRequiresProductID(t *testing.T) {
	p := validProvisioningProfile()
	p.ProductID = uuid.Nil

	assertInvalid(t, p.Validate())
}

func TestProvisioningProfileValidateRequiresVendor(t *testing.T) {
	p := validProvisioningProfile()
	p.Vendor = ""

	assertInvalid(t, p.Validate())
}

func TestProvisioningProfileValidateRequiresProfileName(t *testing.T) {
	p := validProvisioningProfile()
	p.ProfileName = ""

	assertInvalid(t, p.Validate())
}

func TestProvisioningProfileValidateDescriptionIsOptional(t *testing.T) {
	p := validProvisioningProfile() // no description set
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	p.Description = "500 Mbps residential rate-limit + VLAN profile"
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
