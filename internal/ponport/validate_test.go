package ponport_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// assertInvalid mirrors internal/olt/validate_test.go's helper of the
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

func validPONPort() ponport.PONPort {
	return ponport.PONPort{
		OLTID:      uuid.New(),
		PortNumber: 1,
	}
}

func TestPONPortValidate(t *testing.T) {
	if err := validPONPort().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, ponport.PONPort{}.Validate())
}

func TestPONPortValidateRequiresOLTID(t *testing.T) {
	p := validPONPort()
	p.OLTID = uuid.Nil

	assertInvalid(t, p.Validate())
}

func TestPONPortValidateRequiresPositivePortNumber(t *testing.T) {
	zero := validPONPort()
	zero.PortNumber = 0
	assertInvalid(t, zero.Validate())

	negative := validPONPort()
	negative.PortNumber = -1
	assertInvalid(t, negative.Validate())

	for _, n := range []int{1, 2, 16, 128} {
		p := validPONPort()
		p.PortNumber = n
		if err := p.Validate(); err != nil {
			t.Errorf("Validate() (port number %d) = %v, want nil", n, err)
		}
	}
}

func TestPONPortValidateDescriptionIsOptional(t *testing.T) {
	p := validPONPort() // no description set
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	p.Description = "Feeds the north neighborhood splitter"
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
