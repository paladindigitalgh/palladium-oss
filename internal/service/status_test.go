package service_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/service"
)

func TestServiceStatusValidAcceptsDefinedValues(t *testing.T) {
	defined := []service.ServiceStatus{
		service.ServiceStatusPending,
		service.ServiceStatusActive,
		service.ServiceStatusSuspended,
		service.ServiceStatusDisconnected,
	}

	for _, s := range defined {
		if !s.Valid() {
			t.Errorf("%q.Valid() = false, want true", s)
		}
	}
}

func TestServiceStatusValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []service.ServiceStatus{
		"",          // zero value: there is no default status
		"pending",   // wrong case
		"ACTIVE",    // wrong case
		"Cancelled", // not a defined status for Service
	}

	for _, s := range cases {
		if s.Valid() {
			t.Errorf("%q.Valid() = true, want false", s)
		}
	}
}

func TestServiceStatusStringReturnsUnderlyingValue(t *testing.T) {
	if got := service.ServiceStatusSuspended.String(); got != "Suspended" {
		t.Errorf("String() = %q, want %q", got, "Suspended")
	}
}
