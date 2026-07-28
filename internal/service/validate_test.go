package service_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
)

// assertInvalid mirrors internal/product/validate_test.go's helper of the
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

func validService() service.Service {
	return service.Service{
		LocationID:       uuid.New(),
		ProductID:        uuid.New(),
		ServiceProfileID: uuid.New(),
		Status:           service.ServiceStatusPending,
	}
}

func TestServiceValidate(t *testing.T) {
	if err := validService().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, service.Service{}.Validate())
}

func TestServiceValidateRequiresLocationID(t *testing.T) {
	s := validService()
	s.LocationID = uuid.Nil

	assertInvalid(t, s.Validate())
}

func TestServiceValidateRequiresProductID(t *testing.T) {
	s := validService()
	s.ProductID = uuid.Nil

	assertInvalid(t, s.Validate())
}

func TestServiceValidateRequiresServiceProfileID(t *testing.T) {
	s := validService()
	s.ServiceProfileID = uuid.Nil

	assertInvalid(t, s.Validate())
}

func TestServiceValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validService()
	unrecognized.Status = service.ServiceStatus("Cancelled")
	assertInvalid(t, unrecognized.Validate())

	unset := validService()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, st := range []service.ServiceStatus{
		service.ServiceStatusPending,
		service.ServiceStatusActive,
		service.ServiceStatusSuspended,
		service.ServiceStatusDisconnected,
	} {
		s := validService()
		s.Status = st
		if err := s.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", st, err)
		}
	}
}

func TestServiceValidateDescriptionIsOptional(t *testing.T) {
	s := validService() // no description set
	if err := s.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	s.Description = "Primary residential internet service"
	if err := s.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}

func TestServiceValidateLifecycleTimestampsAreOptional(t *testing.T) {
	s := validService() // no lifecycle timestamps set
	if err := s.Validate(); err != nil {
		t.Errorf("Validate() (no lifecycle timestamps) = %v, want nil", err)
	}
}
