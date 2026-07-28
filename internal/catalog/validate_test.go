package catalog_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/location/validate_test.go's helper of
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

func validCatalog() catalog.ProductCatalog {
	return catalog.ProductCatalog{
		Name:   "Residential",
		Status: catalog.CatalogStatusActive,
	}
}

func TestCatalogValidate(t *testing.T) {
	if err := validCatalog().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, catalog.ProductCatalog{}.Validate())
}

func TestCatalogValidateRequiresName(t *testing.T) {
	c := validCatalog()
	c.Name = ""

	assertInvalid(t, c.Validate())
}

func TestCatalogValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validCatalog()
	unrecognized.Status = catalog.CatalogStatus("Archived")
	assertInvalid(t, unrecognized.Validate())

	unset := validCatalog()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []catalog.CatalogStatus{
		catalog.CatalogStatusActive,
		catalog.CatalogStatusInactive,
	} {
		c := validCatalog()
		c.Status = s
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestCatalogValidateDescriptionIsOptional(t *testing.T) {
	c := validCatalog() // no description set
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	c.Description = "Products sold to residential customers"
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
