package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeONUAuthorizer is an in-memory onuAuthorizer.
type fakeONUAuthorizer struct {
	iface string
	err   error

	gotOLTID        uuid.UUID
	gotPort         string
	gotSerialNumber string
}

func (f *fakeONUAuthorizer) AuthorizeONU(_ context.Context, oltID uuid.UUID, port, serialNumber string) (string, error) {
	f.gotOLTID, f.gotPort, f.gotSerialNumber = oltID, port, serialNumber
	if f.err != nil {
		return "", f.err
	}
	return f.iface, nil
}

// fakeDeviceCreator is an in-memory deviceCreator.
type fakeDeviceCreator struct {
	created inventory.Device
	err     error

	gotDevice inventory.Device
	called    bool
}

func (f *fakeDeviceCreator) Create(_ context.Context, device inventory.Device) (inventory.Device, error) {
	f.called = true
	f.gotDevice = device
	if f.err != nil {
		return inventory.Device{}, f.err
	}
	return f.created, nil
}

func TestAuthorizeAndCreateDeviceAuthorizesThenCreates(t *testing.T) {
	oltID := uuid.New()
	deviceModelID := uuid.New()
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/2"}
	deviceID := uuid.New()
	creator := &fakeDeviceCreator{created: inventory.Device{Metadata: inventory.Metadata{ID: deviceID, Name: "New ONU"}}}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator)

	device := inventory.Device{
		Metadata:      inventory.Metadata{Name: "New ONU"},
		DeviceModelID: deviceModelID,
		SerialNumber:  "ISKT2308DD88",
		Status:        inventory.DeviceStatusInstalled,
	}

	created, iface, err := s.AuthorizeAndCreateDevice(context.Background(), oltID, "xgs/6", device)
	if err != nil {
		t.Fatalf("AuthorizeAndCreateDevice() = %v", err)
	}
	if iface != "xgs/6/2" {
		t.Errorf("iface = %q, want %q", iface, "xgs/6/2")
	}
	if created.ID != deviceID {
		t.Errorf("created.ID = %v, want %v", created.ID, deviceID)
	}

	if authorizer.gotOLTID != oltID || authorizer.gotPort != "xgs/6" || authorizer.gotSerialNumber != "ISKT2308DD88" {
		t.Errorf("AuthorizeONU called with (%v, %q, %q), want (%v, %q, %q)",
			authorizer.gotOLTID, authorizer.gotPort, authorizer.gotSerialNumber,
			oltID, "xgs/6", "ISKT2308DD88")
	}
	if !creator.called {
		t.Fatal("Create was never called")
	}
	if creator.gotDevice.SerialNumber != "ISKT2308DD88" || creator.gotDevice.DeviceModelID != deviceModelID {
		t.Errorf("Create called with %+v", creator.gotDevice)
	}
}

func TestAuthorizeAndCreateDeviceDoesNotCreateWhenAuthorizeFails(t *testing.T) {
	authorizer := &fakeONUAuthorizer{err: apperror.Unavailable("could not reach OLT", errors.New("dial failed"))}
	creator := &fakeDeviceCreator{}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator)

	_, _, err := s.AuthorizeAndCreateDevice(context.Background(), uuid.New(), "xgs/6", inventory.Device{SerialNumber: "ISKT2308DD88"})
	if !apperror.Is(err, apperror.KindUnavailable) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindUnavailable)
	}
	if creator.called {
		t.Error("Create was called despite AuthorizeONU failing")
	}
}

func TestAuthorizeAndCreateDevicePropagatesCreateFailureAfterAuthorizeSucceeds(t *testing.T) {
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/2"}
	creator := &fakeDeviceCreator{err: apperror.Conflict("a device with this serial number already exists")}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator)

	_, iface, err := s.AuthorizeAndCreateDevice(context.Background(), uuid.New(), "xgs/6", inventory.Device{SerialNumber: "ISKT2308DD88"})
	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	// The OLT command already ran and cannot be rolled back from here --
	// the caller still learns which interface it landed on so it can
	// reconcile manually (see the service's own doc comment).
	if iface != "xgs/6/2" {
		t.Errorf("iface = %q, want %q even though Create failed", iface, "xgs/6/2")
	}
	if !creator.called {
		t.Error("Create was never called")
	}
}
