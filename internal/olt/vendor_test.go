package olt_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
)

func TestVendorValidAcceptsDefinedValues(t *testing.T) {
	defined := []olt.Vendor{
		olt.VendorKontron,
		olt.VendorNokia,
		olt.VendorCalix,
		olt.VendorAdtran,
		olt.VendorOther,
	}

	for _, v := range defined {
		if !v.Valid() {
			t.Errorf("%q.Valid() = false, want true", v)
		}
	}
}

func TestVendorValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []olt.Vendor{
		"",         // zero value: there is no default vendor
		"kontron",  // wrong case
		"NOKIA",    // wrong case
		"MikroTik", // not a defined vendor for OLT
	}

	for _, v := range cases {
		if v.Valid() {
			t.Errorf("%q.Valid() = true, want false", v)
		}
	}
}

func TestVendorStringReturnsUnderlyingValue(t *testing.T) {
	if got := olt.VendorKontron.String(); got != "Kontron" {
		t.Errorf("String() = %q, want %q", got, "Kontron")
	}
}
