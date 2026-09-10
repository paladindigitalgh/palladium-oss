package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
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

// fakePONPortFinderCreator is an in-memory ponPortFinderCreator. Not
// found by default, so a happy-path test exercising syncAccessTopology
// takes the find-then-create branch unless a test pre-seeds byPort.
type fakePONPortFinderCreator struct {
	byPort map[int]ponport.PONPort

	getErr       error
	createErr    error
	createCalled bool
	gotCreate    ponport.PONPort
}

func (f *fakePONPortFinderCreator) GetByOLTIDAndPortNumber(_ context.Context, _ uuid.UUID, portNumber int) (ponport.PONPort, error) {
	if f.getErr != nil {
		return ponport.PONPort{}, f.getErr
	}
	if p, ok := f.byPort[portNumber]; ok {
		return p, nil
	}
	return ponport.PONPort{}, apperror.NotFound("pon port not found")
}

func (f *fakePONPortFinderCreator) Create(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	f.createCalled = true
	if f.createErr != nil {
		return ponport.PONPort{}, f.createErr
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.gotCreate = p
	return p, nil
}

// fakeAccessInterfaceFinderCreator is an in-memory
// accessInterfaceFinderCreator. Not found by default, so a happy-path
// test exercising syncAccessTopology takes the find-then-create branch
// unless a test pre-seeds byName.
type fakeAccessInterfaceFinderCreator struct {
	byName map[string]accessinterface.AccessInterface

	getErr       error
	createErr    error
	createCalled bool
	gotCreate    accessinterface.AccessInterface
}

func (f *fakeAccessInterfaceFinderCreator) GetByOLTIDAndName(_ context.Context, _ uuid.UUID, name string) (accessinterface.AccessInterface, error) {
	if f.getErr != nil {
		return accessinterface.AccessInterface{}, f.getErr
	}
	if a, ok := f.byName[name]; ok {
		return a, nil
	}
	return accessinterface.AccessInterface{}, apperror.NotFound("access interface not found")
}

func (f *fakeAccessInterfaceFinderCreator) Create(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	f.createCalled = true
	f.gotCreate = a
	if f.createErr != nil {
		return accessinterface.AccessInterface{}, f.createErr
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return a, nil
}

var authorizeAndCreateDeviceTestNow = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

func TestAuthorizeAndCreateDeviceAuthorizesThenCreatesThenRecordsAuthorization(t *testing.T) {
	oltID := uuid.New()
	deviceModelID := uuid.New()
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/2"}
	deviceID := uuid.New()
	creator := &fakeDeviceCreator{created: inventory.Device{Metadata: inventory.Metadata{ID: deviceID, Name: "New ONU"}}}
	authorizations := &fakeOnuAuthorizationCreator{}
	ponPorts := &fakePONPortFinderCreator{}
	interfaces := &fakeAccessInterfaceFinderCreator{}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, ponPorts, interfaces, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

	device := inventory.Device{
		Metadata:      inventory.Metadata{Name: "New ONU"},
		DeviceModelID: deviceModelID,
		SerialNumber:  "ISKT2308DD88",
		Status:        inventory.DeviceStatusUnused,
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

	if !ponPorts.createCalled {
		t.Fatal("PONPort Create was never called")
	}
	if ponPorts.gotCreate.OLTID != oltID || ponPorts.gotCreate.PortNumber != 6 {
		t.Errorf("PONPort Create called with %+v, want OLTID %v and PortNumber 6", ponPorts.gotCreate, oltID)
	}
	if !interfaces.createCalled {
		t.Fatal("AccessInterface Create was never called")
	}
	if interfaces.gotCreate.Name != "xgs/6/2" || interfaces.gotCreate.Technology != accessinterface.TechnologyXGSPON || interfaces.gotCreate.Status != accessinterface.StatusActive {
		t.Errorf("AccessInterface Create called with %+v", interfaces.gotCreate)
	}
	if interfaces.gotCreate.PONPortID != ponPorts.gotCreate.ID {
		t.Errorf("AccessInterface Create's PONPortID = %v, want the newly created PONPort's ID %v", interfaces.gotCreate.PONPortID, ponPorts.gotCreate.ID)
	}
}

// TestAuthorizeAndCreateDeviceReusesExistingAccessTopology proves
// syncAccessTopology finds-or-creates: when a matching PONPort and
// AccessInterface already exist (e.g. a second Device authorized on a
// port Palladium already knows about), it must reuse them, not create
// duplicates.
func TestAuthorizeAndCreateDeviceReusesExistingAccessTopology(t *testing.T) {
	oltID := uuid.New()
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/3"}
	creator := &fakeDeviceCreator{created: inventory.Device{Metadata: inventory.Metadata{ID: uuid.New()}}}
	authorizations := &fakeOnuAuthorizationCreator{}
	existingPort := ponport.PONPort{ID: uuid.New(), OLTID: oltID, PortNumber: 6}
	existingInterface := accessinterface.AccessInterface{ID: uuid.New(), PONPortID: existingPort.ID, Name: "xgs/6/3", Technology: accessinterface.TechnologyXGSPON, Status: accessinterface.StatusActive}
	ponPorts := &fakePONPortFinderCreator{byPort: map[int]ponport.PONPort{6: existingPort}}
	interfaces := &fakeAccessInterfaceFinderCreator{byName: map[string]accessinterface.AccessInterface{"xgs/6/3": existingInterface}}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, ponPorts, interfaces, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

	if _, _, err := s.AuthorizeAndCreateDevice(context.Background(), oltID, "xgs/6", inventory.Device{SerialNumber: "ISKT2308DD88"}); err != nil {
		t.Fatalf("AuthorizeAndCreateDevice() = %v", err)
	}

	if ponPorts.createCalled {
		t.Error("PONPort Create was called despite a matching PONPort already existing")
	}
	if interfaces.createCalled {
		t.Error("AccessInterface Create was called despite a matching AccessInterface already existing")
	}
}

// TestAuthorizeAndCreateDevicePropagatesAccessTopologySyncFailure proves
// a syncAccessTopology failure after everything else has already
// succeeded is surfaced, not swallowed — mirroring
// TestAuthorizeAndCreateDevicePropagatesAuthorizationRecordFailureAfterDeviceCreated
// for this later step.
func TestAuthorizeAndCreateDevicePropagatesAccessTopologySyncFailure(t *testing.T) {
	deviceID := uuid.New()
	authorizer := &fakeONUAuthorizer{iface: "xgs/6/2"}
	creator := &fakeDeviceCreator{created: inventory.Device{Metadata: inventory.Metadata{ID: deviceID}}}
	authorizations := &fakeOnuAuthorizationCreator{}
	ponPorts := &fakePONPortFinderCreator{createErr: apperror.Internal("create pon port", errors.New("db unavailable"))}
	interfaces := &fakeAccessInterfaceFinderCreator{}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, ponPorts, interfaces, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

	created, iface, err := s.AuthorizeAndCreateDevice(context.Background(), uuid.New(), "xgs/6", inventory.Device{SerialNumber: "ISKT2308DD88"})
	if err == nil {
		t.Fatal("AuthorizeAndCreateDevice() error = nil, want an error")
	}
	// The OLT command, Device Create, and OnuAuthorization Create all
	// already succeeded and cannot be rolled back from here -- the caller
	// still learns what was actually persisted so it can reconcile
	// manually (see the service's own doc comment).
	if created.ID != deviceID {
		t.Errorf("created.ID = %v, want %v even though the topology sync failed", created.ID, deviceID)
	}
	if iface != "xgs/6/2" {
		t.Errorf("iface = %q, want %q even though the topology sync failed", iface, "xgs/6/2")
	}
	if !authorizations.called {
		t.Error("OnuAuthorization Create was never called")
	}
}

func TestAuthorizeAndCreateDeviceDoesNotCreateWhenAuthorizeFails(t *testing.T) {
	authorizer := &fakeONUAuthorizer{err: apperror.Unavailable("could not reach OLT", errors.New("dial failed"))}
	creator := &fakeDeviceCreator{}
	authorizations := &fakeOnuAuthorizationCreator{}
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, nil, nil, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

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
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, nil, nil, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

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
	s := NewAuthorizeAndCreateDeviceService(authorizer, creator, authorizations, nil, nil, clock.NewFrozen(authorizeAndCreateDeviceTestNow))

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
