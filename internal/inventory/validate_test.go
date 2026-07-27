package inventory_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid checks that err is a platform validation error, i.e. an
// *apperror.Error of KindInvalid — the contract every domain package's
// Validate() must honor.
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

func uuidPtr(id uuid.UUID) *uuid.UUID { return &id }

func TestSiteValidate(t *testing.T) {
	valid := inventory.Site{Metadata: inventory.Metadata{Name: "Main Office"}}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, inventory.Site{}.Validate())
}

func TestBuildingValidate(t *testing.T) {
	valid := inventory.Building{
		Metadata: inventory.Metadata{Name: "HQ"},
		SiteID:   uuid.New(),
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, inventory.Building{SiteID: uuid.New()}.Validate())
	assertInvalid(t, inventory.Building{Metadata: inventory.Metadata{Name: "HQ"}}.Validate())
}

func TestRoomValidate(t *testing.T) {
	valid := inventory.Room{
		Metadata:   inventory.Metadata{Name: "Server Room"},
		BuildingID: uuid.New(),
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, inventory.Room{BuildingID: uuid.New()}.Validate())
	assertInvalid(t, inventory.Room{Metadata: inventory.Metadata{Name: "Server Room"}}.Validate())
}

func TestRackValidate(t *testing.T) {
	installed := inventory.Rack{
		Metadata: inventory.Metadata{Name: "Rack 1"},
		RoomID:   uuidPtr(uuid.New()),
	}
	if err := installed.Validate(); err != nil {
		t.Errorf("Validate() (installed) = %v, want nil", err)
	}

	inStorage := inventory.Rack{Metadata: inventory.Metadata{Name: "Rack 1"}}
	if err := inStorage.Validate(); err != nil {
		t.Errorf("Validate() (nil RoomID) = %v, want nil; RoomID must be optional", err)
	}

	assertInvalid(t, inventory.Rack{}.Validate())
}

// validDevice returns a Device that satisfies every required field, as a
// base for tests to mutate one field at a time.
func validDevice() inventory.Device {
	return inventory.Device{
		Metadata:     inventory.Metadata{Name: "Switch 1"},
		Manufacturer: "Acme Corp",
		Model:        "X100",
		SerialNumber: "SN-12345",
		Status:       inventory.DeviceStatusInStock,
	}
}

func TestDeviceValidate(t *testing.T) {
	racked := validDevice()
	racked.RackID = uuidPtr(uuid.New())
	if err := racked.Validate(); err != nil {
		t.Errorf("Validate() (racked) = %v, want nil", err)
	}

	inStorage := validDevice()
	if err := inStorage.Validate(); err != nil {
		t.Errorf("Validate() (nil RackID) = %v, want nil; RackID must be optional", err)
	}

	assertInvalid(t, inventory.Device{}.Validate())
}

func TestDeviceValidateRequiresManufacturerModelSerialNumber(t *testing.T) {
	cases := map[string]func(*inventory.Device){
		"manufacturer": func(d *inventory.Device) { d.Manufacturer = "" },
		"model":        func(d *inventory.Device) { d.Model = "" },
		"serialNumber": func(d *inventory.Device) { d.SerialNumber = "" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := validDevice()
			mutate(&d)
			assertInvalid(t, d.Validate())
		})
	}
}

func TestDeviceValidateAssetTagIsOptional(t *testing.T) {
	withoutTag := validDevice()
	withoutTag.AssetTag = ""
	if err := withoutTag.Validate(); err != nil {
		t.Errorf("Validate() (no asset tag) = %v, want nil", err)
	}

	withTag := validDevice()
	withTag.AssetTag = "AT-001"
	if err := withTag.Validate(); err != nil {
		t.Errorf("Validate() (with asset tag) = %v, want nil", err)
	}
}

func TestDeviceValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validDevice()
	unrecognized.Status = inventory.DeviceStatus("Deployed")
	assertInvalid(t, unrecognized.Validate())

	unset := validDevice()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, status := range []inventory.DeviceStatus{
		inventory.DeviceStatusOrdered,
		inventory.DeviceStatusReceived,
		inventory.DeviceStatusInStock,
		inventory.DeviceStatusInstalled,
		inventory.DeviceStatusMaintenance,
		inventory.DeviceStatusRetired,
		inventory.DeviceStatusDisposed,
	} {
		d := validDevice()
		d.Status = status
		if err := d.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", status, err)
		}
	}
}
