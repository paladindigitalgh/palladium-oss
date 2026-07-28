package product_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

func TestProductCategoryValidAcceptsDefinedValues(t *testing.T) {
	defined := []product.ProductCategory{
		product.ProductCategoryInternet,
		product.ProductCategoryVoice,
		product.ProductCategoryIPTV,
		product.ProductCategoryTransport,
		product.ProductCategoryManagedWiFi,
		product.ProductCategoryOther,
	}

	for _, c := range defined {
		if !c.Valid() {
			t.Errorf("%q.Valid() = false, want true", c)
		}
	}
}

func TestProductCategoryValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []product.ProductCategory{
		"",         // zero value: there is no default category
		"internet", // wrong case
		"VOICE",    // wrong case
		"Ethernet", // not a defined category at all
	}

	for _, c := range cases {
		if c.Valid() {
			t.Errorf("%q.Valid() = true, want false", c)
		}
	}
}

func TestProductCategoryStringReturnsUnderlyingValue(t *testing.T) {
	if got := product.ProductCategoryManagedWiFi.String(); got != "ManagedWiFi" {
		t.Errorf("String() = %q, want %q", got, "ManagedWiFi")
	}
}
