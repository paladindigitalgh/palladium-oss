package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
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

// fakeOnuAuthorizationCreator is an in-memory onuAuthorizationCreator.
type fakeOnuAuthorizationCreator struct {
	err error

	gotAuthorization onuauthorization.OnuAuthorization
	called           bool
}

func (f *fakeOnuAuthorizationCreator) Create(_ context.Context, authorization onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error) {
	f.called = true
	f.gotAuthorization = authorization
	if f.err != nil {
		return onuauthorization.OnuAuthorization{}, f.err
	}
	return authorization, nil
}

var authorizeAndCreateDeviceTestNow = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

func TestAuthorizeAndCreateDeviceAuthorizesThenCreatesThenRecordsAuthorization(t *testing.T) {
	oltID := uuid.New()
	deviceModelID := uuid.New()
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/2"}
	deviceID := uuid.New()
	creator := &fakeDeviceCreator{created: inventory.Device{Metadata: inventory.Metadata{ID: deviceID, Name: "New ONU"}}}
	authorizations := &fakeOnuAuthorizationCreator{}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

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

	if !authorizations.called {
		t.Fatal("OnuAuthorization Create was never called")
	}
	want := onuauthorization.OnuAuthorization{
		DeviceID:     deviceID,
		OLTID:        oltID,
		Interface:    "xgs/6/2",
		AuthorizedAt: authorizeAndCreateDeviceTestNow,
	}
	if authorizations.gotAuthorization != want {
		t.Errorf("OnuAuthorization Create called with %+v, want %+v", authorizations.gotAuthorization, want)
	}
}

func TestAuthorizeAndCreateDeviceDoesNotCreateWhenAuthorizeFails(t *testing.T) {
	authorizer := &fakeONUAuthorizer{err: apperror.Unavailable("could not reach OLT", errors.New("dial failed"))}
	creator := &fakeDeviceCreator{}
	authorizations := &fakeOnuAuthorizationCreator{}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

	_, _, err := s.AuthorizeAndCreateDevice(context.Background(), uuid.New(), "xgs/6", inventory.Device{SerialNumber: "ISKT2308DD88"})
	if !apperror.Is(err, apperror.KindUnavailable) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindUnavailable)
	}
	if creator.called {
		t.Error("Create was called despite AuthorizeONU failing")
	}
	if authorizations.called {
		t.Error("OnuAuthorization Create was called despite AuthorizeONU failing")
	}
}

func TestAuthorizeAndCreateDevicePropagatesCreateFailureAfterAuthorizeSucceeds(t *testing.T) {
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/2"}
	creator := &fakeDeviceCreator{err: apperror.Conflict("a device with this serial number already exists")}
	authorizations := &fakeOnuAuthorizationCreator{}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

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
	if authorizations.called {
		t.Error("OnuAuthorization Create was called despite Device Create failing")
	}
}

func TestAuthorizeAndCreateDevicePropagatesAuthorizationRecordFailureAfterDeviceCreated(t *testing.T) {
	deviceID := uuid.New()
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/2"}
	creator := &fakeDeviceCreator{created: inventory.Device{Metadata: inventory.Metadata{ID: deviceID}}}
	authorizations := &fakeOnuAuthorizationCreator{err: apperror.Internal("create onu authorization", errors.New("db unavailable"))}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

	created, iface, err := s.AuthorizeAndCreateDevice(context.Background(), uuid.New(), "xgs/6", inventory.Device{SerialNumber: "ISKT2308DD88"})
	if err == nil {
		t.Fatal("AuthorizeAndCreateDevice() error = nil, want an error")
	}
	// Both the OLT command and the Device Create already succeeded and
	// cannot be rolled back from here -- the caller still learns what was
	// actually persisted so it can reconcile manually (see the service's
	// own doc comment).
	if created.ID != deviceID {
		t.Errorf("created.ID = %v, want %v even though the authorization record failed to save", created.ID, deviceID)
	}
	if iface != "xgs/6/2" {
		t.Errorf("iface = %q, want %q even though the authorization record failed to save", iface, "xgs/6/2")
	}
	if !authorizations.called {
		t.Error("OnuAuthorization Create was never called")
	}
}
