package inventory_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

func TestDeviceStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []inventory.DeviceStatus{
		inventory.DeviceStatusOrdered,
		inventory.DeviceStatusReceived,
		inventory.DeviceStatusInStock,
		inventory.DeviceStatusInstalled,
		inventory.DeviceStatusMaintenance,
		inventory.DeviceStatusRetired,
		inventory.DeviceStatusDisposed,
	}

	for _, status := range defined {
		if !status.Valid() {
			t.Errorf("%q.Valid() = false, want true", status)
		}
	}
}

func TestDeviceStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []inventory.DeviceStatus{
		"",          // zero value: there is no default status
		"ordered",   // wrong case
		"INSTALLED", // wrong case
		"Deployed",  // not a defined status at all
	}

	for _, status := range cases {
		if status.Valid() {
			t.Errorf("%q.Valid() = true, want false", status)
		}
	}
}

func TestDeviceStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := inventory.DeviceStatusInStock.String(); got != "InStock" {
		t.Errorf("String() = %q, want %q", got, "InStock")
	}
}
