package provider_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
)

// assertInvalid mirrors internal/serviceprofile/validate_test.go's
// helper of the same name: every domain package's Validate() must
// return an *apperror.Error of KindInvalid.
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

func validProvider() provider.Provider {
	return provider.Provider{
		Name:   "Acme Fiber",
		Status: provider.StatusActive,
	}
}

func TestProviderValidate(t *testing.T) {
	if err := validProvider().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, provider.Provider{}.Validate())
}

func TestProviderValidateRequiresName(t *testing.T) {
	p := validProvider()
	p.Name = ""

	assertInvalid(t, p.Validate())
}

func TestProviderValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validProvider()
	unrecognized.Status = provider.Status("Archived")
	assertInvalid(t, unrecognized.Validate())

	unset := validProvider()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []provider.Status{
		provider.StatusActive,
		provider.StatusInactive,
	} {
		p := validProvider()
		p.Status = s
		if err := p.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestProviderValidateDescriptionIsOptional(t *testing.T) {
	p := validProvider() // no description set
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	p.Description = "Retail ISP identity for the wholesale open-access network"
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
