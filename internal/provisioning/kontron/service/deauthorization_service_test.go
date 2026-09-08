package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// fakeServiceEquipmentStore scripts GetActiveByDeviceID and records
// every Update call made to it.
type fakeServiceEquipmentStore struct {
	equipment   serviceequipment.ServiceEquipment
	getErr      error
	updateErr   error
	updateCalls []serviceequipment.ServiceEquipment
	gotDeviceID uuid.UUID
}

func (f *fakeServiceEquipmentStore) GetActiveByDeviceID(_ context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error) {
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

func newTestDeauthorizationService(
	dial dialer,
	equipment serviceEquipmentGetter,
	equipmentSvc serviceEquipmentUpdater,
	locate locator,
	olts oltGetter,
	models oltModelGetter,
	attachments accessAttachmentGetter,
	attachmentsSvc accessAttachmentUpdater,
	c clock.Clock,
) *DeauthorizationService {
	return NewDeauthorizationService(dial, equipment, equipmentSvc, locate, olts, models, attachments, attachmentsSvc, c)
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

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, c)

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

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, c)

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

func TestDeauthorizationServiceErrorsForNonONURole(t *testing.T) {
	deviceID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleRouter}

	equipmentStore := &fakeServiceEquipmentStore{equipment: equipment}
	s := newTestDeauthorizationService(&fakeDialer{}, equipmentStore, equipmentStore, &fakeLocator{}, &fakeOLTGetter{}, &fakeOLTModelGetter{}, &fakeAccessAttachmentStore{}, &fakeAccessAttachmentStore{}, clock.NewFrozen(fixedClockTime))

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

	s := newTestDeauthorizationService(&fakeDialer{}, equipmentStore, equipmentStore, locator, olts, models, &fakeAccessAttachmentStore{}, &fakeAccessAttachmentStore{}, clock.NewFrozen(fixedClockTime))

	_, err := s.DeauthorizeONU(context.Background(), deviceID)
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}

func TestDeauthorizationServicePropagatesNoActiveEquipmentAsNotFound(t *testing.T) {
	equipmentStore := &fakeServiceEquipmentStore{getErr: apperror.NotFound("no active equipment")}
	s := newTestDeauthorizationService(&fakeDialer{}, equipmentStore, equipmentStore, &fakeLocator{}, &fakeOLTGetter{}, &fakeOLTModelGetter{}, &fakeAccessAttachmentStore{}, &fakeAccessAttachmentStore{}, clock.NewFrozen(fixedClockTime))

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

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, clock.NewFrozen(fixedClockTime))

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

	s := newTestDeauthorizationService(dialer, equipmentStore, equipmentStore, locator, olts, models, attachmentStore, attachmentStore, clock.NewFrozen(fixedClockTime))

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
