package catalog_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
)

func TestCatalogStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []catalog.CatalogStatus{
		catalog.CatalogStatusActive,
		catalog.CatalogStatusInactive,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestCatalogStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []catalog.CatalogStatus{
		"",         // zero value: there is no default status
		"active",   // wrong case
		"INACTIVE", // wrong case
		"Archived", // not a defined status for Catalog
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestCatalogStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := catalog.CatalogStatusInactive.String(); got != "Inactive" {
		t.Errorf("String() = %q, want %q", got, "Inactive")
	}
}
