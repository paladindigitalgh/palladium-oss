package oltmodel_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/provider/validate_test.go's helper of
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

func validOLTModel() oltmodel.OLTModel {
	return oltmodel.OLTModel{
		Vendor:       oltmodel.VendorKontron,
		Name:         "C16",
		PONPortCount: 16,
	}
}

func TestOLTModelValidate(t *testing.T) {
	if err := validOLTModel().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, oltmodel.OLTModel{}.Validate())
}

func TestOLTModelValidateRequiresName(t *testing.T) {
	m := validOLTModel()
	m.Name = ""

	assertInvalid(t, m.Validate())
}

func TestOLTModelValidateRequiresKnownVendor(t *testing.T) {
	unrecognized := validOLTModel()
	unrecognized.Vendor = oltmodel.Vendor("Huawei")
	assertInvalid(t, unrecognized.Validate())

	unset := validOLTModel()
	unset.Vendor = ""
	assertInvalid(t, unset.Validate())

	for _, v := range []oltmodel.Vendor{
		oltmodel.VendorKontron,
		oltmodel.VendorNokia,
		oltmodel.VendorCalix,
		oltmodel.VendorAdtran,
		oltmodel.VendorOther,
	} {
		m := validOLTModel()
		m.Vendor = v
		if err := m.Validate(); err != nil {
			t.Errorf("Validate() (vendor %q) = %v, want nil", v, err)
		}
	}
}

func TestOLTModelValidateRequiresPositivePONPortCount(t *testing.T) {
	zero := validOLTModel()
	zero.PONPortCount = 0
	assertInvalid(t, zero.Validate())

	negative := validOLTModel()
	negative.PONPortCount = -1
	assertInvalid(t, negative.Validate())

	for _, count := range []int{1, 8, 16, 32} {
		m := validOLTModel()
		m.PONPortCount = count
		if err := m.Validate(); err != nil {
			t.Errorf("Validate() (pon_port_count %d) = %v, want nil", count, err)
		}
	}
}

func TestOLTModelValidateDescriptionIsOptional(t *testing.T) {
	m := validOLTModel() // no description set
	if err := m.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	m.Description = "16-port GPON OLT chassis"
	if err := m.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
