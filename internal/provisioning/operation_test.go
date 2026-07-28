package provisioning_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

func TestProvisioningOperationValidAcceptsDefinedValues(t *testing.T) {
	defined := []provisioning.ProvisioningOperation{
		provisioning.ProvisioningOperationProvision,
		provisioning.ProvisioningOperationReprovision,
		provisioning.ProvisioningOperationSuspend,
		provisioning.ProvisioningOperationResume,
		provisioning.ProvisioningOperationDisconnect,
		provisioning.ProvisioningOperationSynchronize,
	}

	for _, o := range defined {
		if !o.Valid() {
			t.Errorf("%q.Valid() = false, want true", o)
		}
	}
}

func TestProvisioningOperationValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []provisioning.ProvisioningOperation{
		"",            // zero value: there is no default operation
		"provision",   // wrong case
		"SUSPEND",     // wrong case
		"Deprovision", // not a defined operation at all
	}

	for _, o := range cases {
		if o.Valid() {
			t.Errorf("%q.Valid() = true, want false", o)
		}
	}
}

func TestProvisioningOperationStringReturnsUnderlyingValue(t *testing.T) {
	if got := provisioning.ProvisioningOperationSynchronize.String(); got != "Synchronize" {
		t.Errorf("String() = %q, want %q", got, "Synchronize")
	}
}
