package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice/service"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// fakeDeviceGetter is an in-memory deviceGetter, mirroring
// internal/serviceequipment/service's own fakeDeviceStore: Get defaults
// to a Device with DeviceStatusUnused (the real-world default for any
// Device this test file didn't pre-register) when none was explicitly
// seeded for that ID.
type fakeDeviceGetter struct {
	byID map[uuid.UUID]inventory.Device
}

func newFakeDeviceGetter(devices ...inventory.Device) *fakeDeviceGetter {
	f := &fakeDeviceGetter{byID: make(map[uuid.UUID]inventory.Device)}
	for _, d := range devices {
		f.byID[d.ID] = d
	}
	return f
}

func (f *fakeDeviceGetter) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	d, ok := f.byID[id]
	if !ok {
		d = inventory.Device{Metadata: inventory.Metadata{ID: id}, Status: inventory.DeviceStatusUnused}
	}
	return d, nil
}

// fakeServiceEquipmentGetter is an in-memory activeServiceEquipmentGetter.
type fakeServiceEquipmentGetter struct {
	activeByDeviceID map[uuid.UUID]serviceequipment.ServiceEquipment
}

func newFakeServiceEquipmentGetter() *fakeServiceEquipmentGetter {
	return &fakeServiceEquipmentGetter{activeByDeviceID: make(map[uuid.UUID]serviceequipment.ServiceEquipment)}
}

func (f *fakeServiceEquipmentGetter) GetActiveByDeviceID(_ context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	e, ok := f.activeByDeviceID[deviceID]
	if !ok {
		return serviceequipment.ServiceEquipment{}, apperror.NotFound("no active service equipment assignment for device")
	}
	return e, nil
}

// fakeCustomerDeviceRepository is an in-memory
// customerdevice.CustomerDeviceRepository.
type fakeCustomerDeviceRepository struct {
	byID         map[uuid.UUID]customerdevice.CustomerDevice
	createCalled bool
	updateCalled bool
}

func newFakeCustomerDeviceRepository(records ...customerdevice.CustomerDevice) *fakeCustomerDeviceRepository {
	f := &fakeCustomerDeviceRepository{byID: make(map[uuid.UUID]customerdevice.CustomerDevice)}
	for _, r := range records {
		f.byID[r.ID] = r
	}
	return f
}

func (f *fakeCustomerDeviceRepository) Get(_ context.Context, id uuid.UUID) (customerdevice.CustomerDevice, error) {
	r, ok := f.byID[id]
	if !ok {
		return customerdevice.CustomerDevice{}, apperror.NotFound("customer device not found")
	}
	return r, nil
}

func (f *fakeCustomerDeviceRepository) List(_ context.Context) ([]customerdevice.CustomerDevice, error) {
	records := make([]customerdevice.CustomerDevice, 0, len(f.byID))
	for _, r := range f.byID {
		records = append(records, r)
	}
	return records, nil
}

func (f *fakeCustomerDeviceRepository) Create(_ context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	f.createCalled = true
	if cd.ID == uuid.Nil {
		cd.ID = uuid.New()
	}
	f.byID[cd.ID] = cd
	return cd, nil
}

func (f *fakeCustomerDeviceRepository) Update(_ context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	f.updateCalled = true
	if _, ok := f.byID[cd.ID]; !ok {
		return customerdevice.CustomerDevice{}, apperror.NotFound("customer device not found")
	}
	f.byID[cd.ID] = cd
	return cd, nil
}

func (f *fakeCustomerDeviceRepository) GetActiveByDeviceID(_ context.Context, deviceID uuid.UUID) (customerdevice.CustomerDevice, error) {
	for _, r := range f.byID {
		if r.DeviceID == deviceID && r.Active() {
			return r, nil
		}
	}
	return customerdevice.CustomerDevice{}, apperror.NotFound("no active customer device attachment for device")
}

var _ customerdevice.CustomerDeviceRepository = (*fakeCustomerDeviceRepository)(nil)

func validCustomerDevice() customerdevice.CustomerDevice {
	return customerdevice.CustomerDevice{
		CustomerID: uuid.New(),
		DeviceID:   uuid.New(),
	}
}

func TestCustomerDeviceServiceCreateSucceeds(t *testing.T) {
	repo := newFakeCustomerDeviceRepository()
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	created, err := svc.Create(context.Background(), validCustomerDevice())
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if !repo.createCalled {
		t.Error("repository Create() was never called")
	}
}

func TestCustomerDeviceServiceCreateRejectsInvalidRecordWithoutPersisting(t *testing.T) {
	repo := newFakeCustomerDeviceRepository()
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	_, err := svc.Create(context.Background(), customerdevice.CustomerDevice{}) // no CustomerID, DeviceID

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestCustomerDeviceServiceCreateRejectsSecondActiveAssignmentForSameDevice(t *testing.T) {
	deviceID := uuid.New()
	existing := validCustomerDevice()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	repo := newFakeCustomerDeviceRepository(existing)
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	second := validCustomerDevice()
	second.DeviceID = deviceID // same device, still active

	_, err := svc.Create(context.Background(), second)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite an existing active attachment; the uniqueness check must run first")
	}
}

func TestCustomerDeviceServiceCreateAllowsHistoricalReattachment(t *testing.T) {
	deviceID := uuid.New()
	detachedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	historical := validCustomerDevice()
	historical.ID = uuid.New()
	historical.DeviceID = deviceID
	historical.DetachedAt = &detachedAt // no longer active
	repo := newFakeCustomerDeviceRepository(historical)
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	replacement := validCustomerDevice()
	replacement.DeviceID = deviceID

	created, err := svc.Create(context.Background(), replacement)
	if err != nil {
		t.Fatalf("Create() = %v, want success since the existing attachment for this device is historical", err)
	}
	if !created.Active() {
		t.Error("created attachment is not active, want an active new attachment")
	}
}

func TestCustomerDeviceServiceCreateRejectsRetiredDevice(t *testing.T) {
	deviceID := uuid.New()
	devices := newFakeDeviceGetter(inventory.Device{
		Metadata: inventory.Metadata{ID: deviceID},
		Status:   inventory.DeviceStatusRetired,
	})
	repo := newFakeCustomerDeviceRepository()
	svc := service.NewCustomerDeviceService(repo, devices, newFakeServiceEquipmentGetter())

	cd := validCustomerDevice()
	cd.DeviceID = deviceID

	_, err := svc.Create(context.Background(), cd)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite a retired device; the check must run first")
	}
}

func TestCustomerDeviceServiceUpdateSucceeds(t *testing.T) {
	existing := validCustomerDevice()
	existing.ID = uuid.New()
	repo := newFakeCustomerDeviceRepository(existing)
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	toUpdate := existing
	toUpdate.Description = "Updated description"

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", updated.Description, "Updated description")
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

// TestCustomerDeviceServiceUpdateDetachSucceedsWithoutActiveService proves
// the normal detach path: no active ServiceEquipment for the device, so
// the transition to DetachedAt set is allowed.
func TestCustomerDeviceServiceUpdateDetachSucceedsWithoutActiveService(t *testing.T) {
	existing := validCustomerDevice()
	existing.ID = uuid.New()
	repo := newFakeCustomerDeviceRepository(existing)
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	now := time.Now()
	toDetach := existing
	toDetach.DetachedAt = &now

	updated, err := svc.Update(context.Background(), toDetach)
	if err != nil {
		t.Fatalf("Update() = %v, want success detaching a device with no active service", err)
	}
	if updated.Active() {
		t.Error("Active() = true after detaching, want false")
	}
}

// TestCustomerDeviceServiceUpdateDetachBlockedByActiveService is the
// central proof of the "block, don't cascade" decision: detaching a
// Device that still fulfills an active Service must fail with a
// Conflict, and must not persist the detach.
func TestCustomerDeviceServiceUpdateDetachBlockedByActiveService(t *testing.T) {
	deviceID := uuid.New()
	existing := validCustomerDevice()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	repo := newFakeCustomerDeviceRepository(existing)

	equipment := newFakeServiceEquipmentGetter()
	equipment.activeByDeviceID[deviceID] = serviceequipment.ServiceEquipment{
		ID: uuid.New(), ServiceID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU,
	}

	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), equipment)

	now := time.Now()
	toDetach := existing
	toDetach.DetachedAt = &now

	_, err := svc.Update(context.Background(), toDetach)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}

	stored, getErr := repo.Get(context.Background(), existing.ID)
	if getErr != nil {
		t.Fatalf("Get() = %v", getErr)
	}
	if !stored.Active() {
		t.Error("the stored record was detached despite the block; Update must not have persisted the change")
	}
}

// TestCustomerDeviceServiceUpdateOfAlreadyDetachedRecordDoesNotCheckService
// proves the service-equipment check only fires on the active-to-detached
// transition itself, not on every subsequent edit to an already-detached
// record.
func TestCustomerDeviceServiceUpdateOfAlreadyDetachedRecordDoesNotCheckService(t *testing.T) {
	deviceID := uuid.New()
	detachedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	existing := validCustomerDevice()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	existing.DetachedAt = &detachedAt
	repo := newFakeCustomerDeviceRepository(existing)

	equipment := newFakeServiceEquipmentGetter()
	equipment.activeByDeviceID[deviceID] = serviceequipment.ServiceEquipment{
		ID: uuid.New(), ServiceID: uuid.New(), DeviceID: deviceID, Role: serviceequipment.EquipmentRoleONU,
	}

	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), equipment)

	toUpdate := existing
	toUpdate.Description = "editing history, not detaching anything new"

	if _, err := svc.Update(context.Background(), toUpdate); err != nil {
		t.Fatalf("Update() = %v, want success (record was already detached before this edit)", err)
	}
}

func TestCustomerDeviceServiceUpdateRejectsReassigningDeviceWithExistingActiveAssignment(t *testing.T) {
	busyDeviceID := uuid.New()
	busy := validCustomerDevice()
	busy.ID = uuid.New()
	busy.DeviceID = busyDeviceID

	toReassign := validCustomerDevice()
	toReassign.ID = uuid.New()
	repo := newFakeCustomerDeviceRepository(busy, toReassign)
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	toReassign.DeviceID = busyDeviceID // now targets a device that's already actively attached

	_, err := svc.Update(context.Background(), toReassign)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestCustomerDeviceServiceUpdateRejectsInvalidRecordWithoutPersisting(t *testing.T) {
	existing := validCustomerDevice()
	existing.ID = uuid.New()
	repo := newFakeCustomerDeviceRepository(existing)
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	invalid := existing
	invalid.DeviceID = uuid.Nil

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestCustomerDeviceServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeCustomerDeviceRepository()
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestCustomerDeviceServiceListDelegatesToRepository(t *testing.T) {
	a := validCustomerDevice()
	a.ID = uuid.New()
	b := validCustomerDevice()
	b.ID = uuid.New()
	repo := newFakeCustomerDeviceRepository(a, b)
	svc := service.NewCustomerDeviceService(repo, newFakeDeviceGetter(), newFakeServiceEquipmentGetter())

	records, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(records))
	}
}
