package onuauthorization_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/accessattachment/validate_test.go's
// helper of the same name.
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

func validOnuAuthorization() onuauthorization.OnuAuthorization {
	return onuauthorization.OnuAuthorization{
		DeviceID:     uuid.New(),
		OLTID:        uuid.New(),
		Interface:    "xgs/1/3",
		AuthorizedAt: time.Now(),
	}
}

func TestOnuAuthorizationValidate(t *testing.T) {
	if err := validOnuAuthorization().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, onuauthorization.OnuAuthorization{}.Validate())
}

func TestOnuAuthorizationValidateRequiresDeviceID(t *testing.T) {
	a := validOnuAuthorization()
	a.DeviceID = uuid.Nil

	assertInvalid(t, a.Validate())
}

func TestOnuAuthorizationValidateRequiresOLTID(t *testing.T) {
	a := validOnuAuthorization()
	a.OLTID = uuid.Nil

	assertInvalid(t, a.Validate())
}

func TestOnuAuthorizationValidateRequiresInterface(t *testing.T) {
	a := validOnuAuthorization()
	a.Interface = ""

	assertInvalid(t, a.Validate())
}

func TestOnuAuthorizationValidateRequiresAuthorizedAt(t *testing.T) {
	a := validOnuAuthorization()
	a.AuthorizedAt = time.Time{}

	assertInvalid(t, a.Validate())
}

func TestOnuAuthorizationValidateDeauthorizedAtIsOptional(t *testing.T) {
	a := validOnuAuthorization() // no DeauthorizedAt set
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (no deauthorized_at) = %v, want nil", err)
	}
}

func TestOnuAuthorizationActive(t *testing.T) {
	a := validOnuAuthorization()
	if !a.Active() {
		t.Error("Active() = false for an OnuAuthorization with no DeauthorizedAt, want true")
	}

	deauthorizedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	a.DeauthorizedAt = &deauthorizedAt
	if a.Active() {
		t.Error("Active() = true for an OnuAuthorization with DeauthorizedAt set, want false")
	}
}
