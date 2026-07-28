package product_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

// assertInvalid mirrors internal/catalog/validate_test.go's helper of the
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

func validProduct() product.Product {
	return product.Product{
		CatalogID: uuid.New(),
		Name:      "Residential Internet 100/20",
		Category:  product.ProductCategoryInternet,
		Status:    product.ProductStatusActive,
	}
}

func TestProductValidate(t *testing.T) {
	if err := validProduct().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, product.Product{}.Validate())
}

func TestProductValidateRequiresCatalogID(t *testing.T) {
	p := validProduct()
	p.CatalogID = uuid.Nil

	assertInvalid(t, p.Validate())
}

func TestProductValidateRequiresName(t *testing.T) {
	p := validProduct()
	p.Name = ""

	assertInvalid(t, p.Validate())
}

func TestProductValidateRequiresKnownCategory(t *testing.T) {
	unrecognized := validProduct()
	unrecognized.Category = product.ProductCategory("Ethernet")
	assertInvalid(t, unrecognized.Validate())

	unset := validProduct()
	unset.Category = ""
	assertInvalid(t, unset.Validate())

	for _, c := range []product.ProductCategory{
		product.ProductCategoryInternet,
		product.ProductCategoryVoice,
		product.ProductCategoryIPTV,
		product.ProductCategoryTransport,
		product.ProductCategoryManagedWiFi,
		product.ProductCategoryOther,
	} {
		p := validProduct()
		p.Category = c
		if err := p.Validate(); err != nil {
			t.Errorf("Validate() (category %q) = %v, want nil", c, err)
		}
	}
}

func TestProductValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validProduct()
	unrecognized.Status = product.ProductStatus("Inactive")
	assertInvalid(t, unrecognized.Validate())

	unset := validProduct()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []product.ProductStatus{
		product.ProductStatusActive,
		product.ProductStatusRetired,
	} {
		p := validProduct()
		p.Status = s
		if err := p.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestProductValidateDescriptionIsOptional(t *testing.T) {
	p := validProduct() // no description set
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	p.Description = "100 Mbps down / 20 Mbps up residential internet"
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
