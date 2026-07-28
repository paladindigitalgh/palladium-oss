package provisioning_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

func TestProvisioningStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusPending,
		provisioning.ProvisioningStatusRunning,
		provisioning.ProvisioningStatusSucceeded,
		provisioning.ProvisioningStatusFailed,
		provisioning.ProvisioningStatusCancelled,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestProvisioningStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []provisioning.ProvisioningStatus{
		"",           // zero value: there is no default status
		"pending",    // wrong case
		"RUNNING",    // wrong case
		"Cancelling", // not a defined status at all
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestProvisioningStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := provisioning.ProvisioningStatusFailed.String(); got != "Failed" {
		t.Errorf("String() = %q, want %q", got, "Failed")
	}
}

// TestProvisioningStatusCanTransitionToAllowsSpecifiedTransitions proves
// every transition this milestone's goal 7 lists as allowed really is.
func TestProvisioningStatusCanTransitionToAllowsSpecifiedTransitions(t *testing.T) {
	cases := []struct {
		from, to provisioning.ProvisioningStatus
	}{
		{provisioning.ProvisioningStatusPending, provisioning.ProvisioningStatusRunning},
		{provisioning.ProvisioningStatusPending, provisioning.ProvisioningStatusCancelled},
		{provisioning.ProvisioningStatusRunning, provisioning.ProvisioningStatusSucceeded},
		{provisioning.ProvisioningStatusRunning, provisioning.ProvisioningStatusFailed},
		{provisioning.ProvisioningStatusRunning, provisioning.ProvisioningStatusCancelled},
		{provisioning.ProvisioningStatusFailed, provisioning.ProvisioningStatusPending},
	}

	for _, c := range cases {
		if !c.from.CanTransitionTo(c.to) {
			t.Errorf("%q.CanTransitionTo(%q) = false, want true", c.from, c.to)
		}
	}
}

// TestProvisioningStatusCanTransitionToRejectsEverythingElse proves every
// transition NOT listed in goal 7 is rejected, including from the two
// terminal statuses (Succeeded, Cancelled) and self-transitions.
func TestProvisioningStatusCanTransitionToRejectsEverythingElse(t *testing.T) {
	all := []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusPending,
		provisioning.ProvisioningStatusRunning,
		provisioning.ProvisioningStatusSucceeded,
		provisioning.ProvisioningStatusFailed,
		provisioning.ProvisioningStatusCancelled,
	}

	allowed := map[provisioning.ProvisioningStatus]map[provisioning.ProvisioningStatus]bool{
		provisioning.ProvisioningStatusPending: {
			provisioning.ProvisioningStatusRunning:   true,
			provisioning.ProvisioningStatusCancelled: true,
		},
		provisioning.ProvisioningStatusRunning: {
			provisioning.ProvisioningStatusSucceeded: true,
			provisioning.ProvisioningStatusFailed:    true,
			provisioning.ProvisioningStatusCancelled: true,
		},
		provisioning.ProvisioningStatusFailed: {
			provisioning.ProvisioningStatusPending: true,
		},
	}

	for _, from := range all {
		for _, to := range all {
			want := allowed[from][to]
			if got := from.CanTransitionTo(to); got != want {
				t.Errorf("%q.CanTransitionTo(%q) = %v, want %v", from, to, got, want)
			}
		}
	}
}
