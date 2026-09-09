package inventory_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

func TestDeviceStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []inventory.DeviceStatus{
		inventory.DeviceStatusUnused,
		inventory.DeviceStatusActive,
		inventory.DeviceStatusRetired,
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
		"active",    // wrong case
		"UNUSED",    // wrong case
		"InStock",   // a status this domain used to have, no longer defined
		"Installed", // same
	}

	for _, status := range cases {
		if status.Valid() {
			t.Errorf("%q.Valid() = true, want false", status)
		}
	}
}

func TestDeviceStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := inventory.DeviceStatusUnused.String(); got != "Unused" {
		t.Errorf("String() = %q, want %q", got, "Unused")
	}
}
