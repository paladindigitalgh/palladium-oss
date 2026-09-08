package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// fakeServiceEquipmentStore scripts GetLatestByDeviceID and records
// every Update call made to it.
type fakeServiceEquipmentStore struct {
	equipment   serviceequipment.ServiceEquipment
	getErr      error
	updateErr   error
	updateCalls []serviceequipment.ServiceEquipment
	gotDeviceID uuid.UUID
}

func (f *fakeServiceEquipmentStore) GetLatestByDeviceID(_ context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	f.gotDeviceID = deviceID
	if f.getErr != nil {
		return serviceequipment.ServiceEquipment{}, f.getErr
	}
	return f.equipment, nil
}

func (f *fakeServiceEquipmentStore) Update(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	f.updateCalls = append(f.updateCalls, e)
	if f.updateErr != nil {
		return serviceequipment.ServiceEquipment{}, f.updateErr
	}
	return e, nil
}

// fakeAccessAttachmentStore scripts GetActiveByServiceEquipmentID and
// records every Update call made to it.
type fakeAccessAttachmentStore struct {
	attachment  accessattachment.AccessAttachment
	getErr      error
	updateErr   error
	updateCalls []accessattachment.AccessAttachment
}

func (f *fakeAccessAttachmentStore) GetActiveByServiceEquipmentID(_ context.Context, _ uuid.UUID) (accessattachment.AccessAttachment, error) {
	if f.getErr != nil {
		return accessattachment.AccessAttachment{}, f.getErr
	}
	return f.attachment, nil
}

func (f *fakeAccessAttachmentStore) Update(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	f.updateCalls = append(f.updateCalls, a)
	if f.updateErr != nil {
		return accessattachment.AccessAttachment{}, f.updateErr
	}
	return a, nil
}

// fakeDeviceStore scripts Get and records every Update call made to it.
// Defaults to an Installed Device if no device is set, matching the
// common case (DeauthorizeONU is only ever offered for an Installed
// Device — see DeviceDetailView.vue's canDeleteONU).
type fakeDeviceStore struct {
	device      inventory.Device
	getErr      error
	updateErr   error
	updateCalls []inventory.Device
}

func (f *fakeDeviceStore) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	if f.getErr != nil {
		return inventory.Device{}, f.getErr
	}
	d := f.device
	if d.Status == "" {
		d = inventory.Device{Metadata: inventory.Metadata{ID: id}, Status: inventory.DeviceStatusInstalled}
	}
	return d, nil
}

func (f *fakeDeviceStore) Update(_ context.Context, d inventory.Device) (inventory.Device, error) {
	f.updateCalls = append(f.updateCalls, d)
	if f.updateErr != nil {
		return inventory.Device{}, f.updateErr
	}
	return d, nil
}

func newTestDeauthorizationService(
	dial dialer,
	equipment latestServiceEquipmentGetter,
	equipmentSvc serviceEquipmentUpdater,
	locate latestLocator,
	olts oltGetter,
	models oltModelGetter,
	attachments accessAttachmentGetter,
	attachmentsSvc accessAttachmentUpdater,
	devices *fakeDeviceStore,
	c clock.Clock,
) *DeauthorizationService {
	return NewDeauthorizationService(dial, equipment, equipmentSvc, locate, olts, models, attachments, attachmentsSvc, devices, devices, c)
}

var fixedClockTime = time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

func TestDeauthorizationServiceSucceedsAndMarksRecordsRemoved(t *testing.T) {
	deviceID := uuid.New()
	oltID := uuid.New()
	equipmentID := uuid.New()
	attachmentID := uuid.New()

	equipment := serviceequipment.ServiceEquipment{ID: equipmentID, DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU}
	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	attachmentStore := &fakeAccessAttachmentStore{attachment: accessattachment.AccessAttachment{ID: attachmentID, ServiceEquipmentID: equipmentID}}
	c := clock.NewFrozen(fixedClockTime)

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, &fakeDeviceStore{}, c)

	iface, err := s.DeauthorizeONU(context.Background(), deviceID)
	if err != nil {
		t.Fatalf("DeauthorizeONU() = %v", err)
	}
	if iface != "xgs/6/3" {
		t.Errorf("DeauthorizeONU() interface = %q, want %q", iface, "xgs/6/3")
	}
	if equipmentStore.gotDeviceID != deviceID {
		t.Errorf("gotDeviceID = %v, want %v", equipmentStore.gotDeviceID, deviceID)
	}

	want := []string{"configure", "interface xgs/6/3", "no onu serial-number", "exit", "exit", "save config"}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", shell.calls, want)
	}
	if !shell.closeCalled {
		t.Error("shell was not closed")
	}

	if len(attachmentStore.updateCalls) != 1 {
		t.Fatalf("attachment Update calls = %d, want 1", len(attachmentStore.updateCalls))
	}
	if attachmentStore.updateCalls[0].RemovedAt == nil || !attachmentStore.updateCalls[0].RemovedAt.Equal(fixedClockTime) {
		t.Errorf("attachment RemovedAt = %v, want %v", attachmentStore.updateCalls[0].RemovedAt, fixedClockTime)
	}
	if attachmentStore.updateCalls[0].RemovalReason == "" {
		t.Error("attachment RemovalReason was not set")
	}

	if len(equipmentStore.updateCalls) != 1 {
		t.Fatalf("equipment Update calls = %d, want 1", len(equipmentStore.updateCalls))
	}
	if equipmentStore.updateCalls[0].RemovedAt == nil || !equipmentStore.updateCalls[0].RemovedAt.Equal(fixedClockTime) {
		t.Errorf("equipment RemovedAt = %v, want %v", equipmentStore.updateCalls[0].RemovedAt, fixedClockTime)
	}
}

func TestDeauthorizationServiceSucceedsWithNoActiveAttachment(t *testing.T) {
	deviceID := uuid.New()
	oltID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONT}

	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	attachmentStore := &fakeAccessAttachmentStore{getErr: apperror.NotFound("no active attachment")}
	c := clock.NewFrozen(fixedClockTime)

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, &fakeDeviceStore{}, c)

	_, err := s.DeauthorizeONU(context.Background(), deviceID)
	if err != nil {
		t.Fatalf("DeauthorizeONU() = %v", err)
	}
	if len(attachmentStore.updateCalls) != 0 {
		t.Error("attachment Update should not have been called when there is no active attachment")
	}
	if len(equipmentStore.updateCalls) != 1 {
		t.Fatalf("equipment Update calls = %d, want 1", len(equipmentStore.updateCalls))
	}
}

func TestDeauthorizationServiceSucceedsWhenEquipmentAlreadyRemovedByCustomerRemoval(t *testing.T) {
	deviceID := uuid.New()
	oltID := uuid.New()
	equipmentID := uuid.New()
	removedAt := fixedClockTime.Add(-time.Hour)

	// Mirrors what internal/customer/removal.RemovalService leaves behind:
	// ServiceEquipment (and its AccessAttachment) already marked removed
	// to untie the Device from the Customer, while the ONU itself is
	// still authorized on the real OLT and needs Delete ONU to finish the
	// job.
	equipment := serviceequipment.ServiceEquipment{
		ID: equipmentID, DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU,
		RemovedAt: &removedAt,
	}
	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/1/2"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	attachmentStore := &fakeAccessAttachmentStore{getErr: apperror.NotFound("no active attachment")}
	c := clock.NewFrozen(fixedClockTime)

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, &fakeDeviceStore{}, c)

	iface, err := s.DeauthorizeONU(context.Background(), deviceID)
	if err != nil {
		t.Fatalf("DeauthorizeONU() = %v, want success even though ServiceEquipment was already removed", err)
	}
	if iface != "xgs/1/2" {
		t.Errorf("DeauthorizeONU() interface = %q, want %q", iface, "xgs/1/2")
	}
	if len(equipmentStore.updateCalls) != 0 {
		t.Errorf("equipment Update calls = %d, want 0 (already removed, nothing new to persist)", len(equipmentStore.updateCalls))
	}
}

func TestDeauthorizationServiceRetiresTheDevice(t *testing.T) {
	deviceID := uuid.New()
	oltID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU}

	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	attachmentStore := &fakeAccessAttachmentStore{getErr: apperror.NotFound("no active attachment")}
	devices := &fakeDeviceStore{device: inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusInstalled}}

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, devices, clock.NewFrozen(fixedClockTime))

	if _, err := s.DeauthorizeONU(context.Background(), deviceID); err != nil {
		t.Fatalf("DeauthorizeONU() = %v", err)
	}
	if len(devices.updateCalls) != 1 {
		t.Fatalf("device Update calls = %d, want 1", len(devices.updateCalls))
	}
	if devices.updateCalls[0].Status != inventory.DeviceStatusRetired {
		t.Errorf("device Status = %q, want %q", devices.updateCalls[0].Status, inventory.DeviceStatusRetired)
	}
}

func TestDeauthorizationServiceDoesNotRewriteAnAlreadyTerminalDeviceStatus(t *testing.T) {
	deviceID := uuid.New()
	oltID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU}

	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	attachmentStore := &fakeAccessAttachmentStore{getErr: apperror.NotFound("no active attachment")}
	devices := &fakeDeviceStore{device: inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusDisposed}}

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, devices, clock.NewFrozen(fixedClockTime))

	if _, err := s.DeauthorizeONU(context.Background(), deviceID); err != nil {
		t.Fatalf("DeauthorizeONU() = %v", err)
	}
	if len(devices.updateCalls) != 0 {
		t.Errorf("device Update calls = %d, want 0 (already Disposed, nothing to retire)", len(devices.updateCalls))
	}
}

func TestDeauthorizationServiceErrorsForNonONURole(t *testing.T) {
	deviceID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleRouter}

	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	s := newTestDeauthorizationService(&fakeDialer{}, equipmentStore, equipmentStore, &fakeLocator{}, &fakeOLTGetter{}, &fakeOLTModelGetter{}, &fakeAccessAttachmentStore{}, &fakeAccessAttachmentStore{}, &fakeDeviceStore{}, clock.NewFrozen(fixedClockTime))

	_, err := s.DeauthorizeONU(context.Background(), deviceID)
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}

func TestDeauthorizationServiceErrorsForNonKontronOLT(t *testing.T) {
	deviceID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU}
	nokiaModel := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorNokia, Name: "ISAM"}

	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: uuid.New(), Interface: "1/1/1"}}
	olts := &fakeOLTGetter{olt: olt.OLT{OLTModelID: nokiaModel.ID}}
	models := &fakeOLTModelGetter{model: nokiaModel}

	s := newTestDeauthorizationService(&fakeDialer{}, equipmentStore, equipmentStore, locator, olts, models, &fakeAccessAttachmentStore{}, &fakeAccessAttachmentStore{}, &fakeDeviceStore{}, clock.NewFrozen(fixedClockTime))

	_, err := s.DeauthorizeONU(context.Background(), deviceID)
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}

func TestDeauthorizationServicePropagatesNoActiveEquipmentAsNotFound(t *testing.T) {
	equipmentStore := &fakeServiceEquipmentStore{getErr: apperror.NotFound("no active equipment")}
	s := newTestDeauthorizationService(&fakeDialer{}, equipmentStore, equipmentStore, &fakeLocator{}, &fakeOLTGetter{}, &fakeOLTModelGetter{}, &fakeAccessAttachmentStore{}, &fakeAccessAttachmentStore{}, &fakeDeviceStore{}, clock.NewFrozen(fixedClockTime))

	_, err := s.DeauthorizeONU(context.Background(), uuid.New())
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestDeauthorizationServiceDoesNotMarkRecordsRemovedWhenCommandFails(t *testing.T) {
	deviceID := uuid.New()
	oltID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU}

	shell := &fakeShell{outputs: map[string]string{"no onu serial-number": "no onu configured on this interface"}}
	dialer := &fakeDialer{shell: shell}
	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	attachmentStore := &fakeAccessAttachmentStore{attachment: accessattachment.AccessAttachment{ID: uuid.New()}}

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, &fakeDeviceStore{}, clock.NewFrozen(fixedClockTime))

	_, err := s.DeauthorizeONU(context.Background(), deviceID)
	if err == nil {
		t.Fatal("DeauthorizeONU() error = nil, want an error")
	}
	if len(attachmentStore.updateCalls) != 0 || len(equipmentStore.updateCalls) != 0 {
		t.Error("records must not be marked removed when the OLT command itself failed")
	}
	if !shell.closeCalled {
		t.Error("shell was not closed after a command failure")
	}
}

func TestDeauthorizationServiceSurfacesErrorWhenRecordUpdateFailsAfterCommandSucceeds(t *testing.T) {
	deviceID := uuid.New()
	oltID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU}

	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment, updateErr: errors.New("db unavailable")}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	attachmentStore := &fakeAccessAttachmentStore{attachment: accessattachment.AccessAttachment{ID: uuid.New()}}

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, &fakeDeviceStore{}, clock.NewFrozen(fixedClockTime))

	// This proves the documented partial-failure edge: the OLT command
	// (scripted to succeed via shell's empty outputs) has already run by
	// the time equipmentStore.Update fails, and that failure is still
	// surfaced rather than swallowed.
	_, err := s.DeauthorizeONU(context.Background(), deviceID)
	if err == nil {
		t.Fatal("DeauthorizeONU() error = nil, want an error from the failed record update")
	}
	if len(attachmentStore.updateCalls) != 1 {
		t.Errorf("attachment Update calls = %d, want 1 (it should still have been attempted)", len(attachmentStore.updateCalls))
	}
}
