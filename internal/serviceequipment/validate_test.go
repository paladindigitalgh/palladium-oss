package serviceequipment_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// assertInvalid mirrors internal/service/validate_test.go's helper of
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

func validServiceEquipment() serviceequipment.ServiceEquipment {
	return serviceequipment.ServiceEquipment{
		ServiceID: uuid.New(),
		DeviceID:  uuid.New(),
		Role:      serviceequipment.EquipmentRoleONU,
	}
}

func TestServiceEquipmentValidate(t *testing.T) {
	if err := validServiceEquipment().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, serviceequipment.ServiceEquipment{}.Validate())
}

func TestServiceEquipmentValidateRequiresServiceID(t *testing.T) {
	e := validServiceEquipment()
	e.ServiceID = uuid.Nil

	assertInvalid(t, e.Validate())
}

func TestServiceEquipmentValidateRequiresDeviceID(t *testing.T) {
	e := validServiceEquipment()
	e.DeviceID = uuid.Nil

	assertInvalid(t, e.Validate())
}

func TestServiceEquipmentValidateRequiresKnownRole(t *testing.T) {
	unrecognized := validServiceEquipment()
	unrecognized.Role = serviceequipment.EquipmentRole("Switch")
	assertInvalid(t, unrecognized.Validate())

	unset := validServiceEquipment()
	unset.Role = ""
	assertInvalid(t, unset.Validate())

	for _, r := range []serviceequipment.EquipmentRole{
		serviceequipment.EquipmentRoleONU,
		serviceequipment.EquipmentRoleGateway,
		serviceequipment.EquipmentRoleRouter,
		serviceequipment.EquipmentRoleONT,
		serviceequipment.EquipmentRoleWiFiAccessPoint,
		serviceequipment.EquipmentRoleUPS,
		serviceequipment.EquipmentRoleOther,
	} {
		e := validServiceEquipment()
		e.Role = r
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() (role %q) = %v, want nil", r, err)
		}
	}
}

func TestServiceEquipmentValidateDescriptionIsOptional(t *testing.T) {
	e := validServiceEquipment() // no description set
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	e.Description = "Installed in the network closet"
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}

func TestServiceEquipmentValidateLifecycleTimestampsAreOptional(t *testing.T) {
	e := validServiceEquipment() // no lifecycle timestamps set
	if err := e.Validate(); err != nil {
		t.Errorf("Validate() (no lifecycle timestamps) = %v, want nil", err)
	}
}

func TestServiceEquipmentActive(t *testing.T) {
	e := validServiceEquipment()
	if !e.Active() {
		t.Error("Active() = false for a ServiceEquipment with no RemovedAt, want true")
	}

	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	e.RemovedAt = &removedAt
	if e.Active() {
		t.Error("Active() = true for a ServiceEquipment with RemovedAt set, want false")
	}
}
