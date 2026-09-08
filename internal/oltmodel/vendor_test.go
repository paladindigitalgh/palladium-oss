package oltmodel_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
)

func TestVendorValidAcceptsDefinedValues(t *testing.T) {
	defined := []oltmodel.Vendor{
		oltmodel.VendorKontron,
		oltmodel.VendorNokia,
		oltmodel.VendorCalix,
		oltmodel.VendorAdtran,
		oltmodel.VendorOther,
	}

	for _, v := range defined {
		if !v.Valid() {
			t.Errorf("%q.Valid() = false, want true", v)
		}
	}
}

func TestVendorValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []oltmodel.Vendor{
		"",         // zero value: there is no default vendor
		"kontron",  // wrong case
		"NOKIA",    // wrong case
		"MikroTik", // not a defined vendor for OLTModel
	}

	for _, v := range cases {
		if v.Valid() {
			t.Errorf("%q.Valid() = true, want false", v)
		}
	}
}

func TestVendorStringReturnsUnderlyingValue(t *testing.T) {
	if got := oltmodel.VendorKontron.String(); got != "Kontron" {
		t.Errorf("String() = %q, want %q", got, "Kontron")
	}
}
