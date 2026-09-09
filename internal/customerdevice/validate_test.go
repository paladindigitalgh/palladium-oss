package customerdevice_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/serviceequipment/validate_test.go's
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

func validCustomerDevice() customerdevice.CustomerDevice {
	return customerdevice.CustomerDevice{
		CustomerID: uuid.New(),
		DeviceID:   uuid.New(),
	}
}

func TestCustomerDeviceValidate(t *testing.T) {
	if err := validCustomerDevice().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, customerdevice.CustomerDevice{}.Validate())
}

func TestCustomerDeviceValidateRequiresCustomerID(t *testing.T) {
	c := validCustomerDevice()
	c.CustomerID = uuid.Nil

	assertInvalid(t, c.Validate())
}

func TestCustomerDeviceValidateRequiresDeviceID(t *testing.T) {
	c := validCustomerDevice()
	c.DeviceID = uuid.Nil

	assertInvalid(t, c.Validate())
}

func TestCustomerDeviceValidateDescriptionIsOptional(t *testing.T) {
	c := validCustomerDevice() // no description set
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	c.Description = "In the basement network closet"
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}

func TestCustomerDeviceValidateLifecycleTimestampsAreOptional(t *testing.T) {
	c := validCustomerDevice() // no lifecycle timestamps set
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() (no lifecycle timestamps) = %v, want nil", err)
	}
}

func TestCustomerDeviceActive(t *testing.T) {
	c := validCustomerDevice()
	if !c.Active() {
		t.Error("Active() = false for a CustomerDevice with no DetachedAt, want true")
	}

	detachedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	c.DetachedAt = &detachedAt
	if c.Active() {
		t.Error("Active() = true for a CustomerDevice with DetachedAt set, want false")
	}
}
