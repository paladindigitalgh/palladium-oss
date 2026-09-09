package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/service"
)

// fakeDeviceStore is an in-memory deviceGetter/deviceUpdater, mirroring
// internal/provisioning/kontron/service's own fakeDeviceStore of the
// same name exactly: Get defaults to a Device with DeviceStatusUnused
// (the real-world default for any Device this test file didn't
// pre-register — see inventory.DeviceStatus's own doc comment) when
// none was explicitly seeded for that ID, so most tests never need to
// register one at all.
type fakeDeviceStore struct {
	byID        map[uuid.UUID]inventory.Device
	updateCalls []inventory.Device
}

func newFakeDeviceStore(devices ...inventory.Device) *fakeDeviceStore {
	f := &fakeDeviceStore{byID: make(map[uuid.UUID]inventory.Device)}
	for _, d := range devices {
		f.byID[d.ID] = d
	}
	return f
}

func (f *fakeDeviceStore) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	d, ok := f.byID[id]
	if !ok {
		d = inventory.Device{Metadata: inventory.Metadata{ID: id}, Status: inventory.DeviceStatusUnused}
	}
	return d, nil
}

func (f *fakeDeviceStore) Update(_ context.Context, d inventory.Device) (inventory.Device, error) {
	f.updateCalls = append(f.updateCalls, d)
	f.byID[d.ID] = d
	return d, nil
}

// fakeServiceEquipmentRepository is an in-memory
// serviceequipment.ServiceEquipmentRepository. Like
// internal/service/service/service_service_test.go's
// fakeServiceRepository, it exists so ServiceEquipmentService's business
// logic — validate, enforce active-assignment uniqueness, then delegate —
// is tested without a real database; the layer below (this domain's
// postgres package) already covers GetActiveByDeviceID's own SQL against
// real PostgreSQL. GetActiveByDeviceID here does the same linear scan a
// real query's WHERE device_id = $1 AND removed_at IS NULL would express
// declaratively, which is exactly what makes it a faithful enough fake
// for testing the service layer's use of it.
type fakeServiceEquipmentRepository struct {
	byID         map[uuid.UUID]serviceequipment.ServiceEquipment
	createCalled bool
	updateCalled bool
}

func newFakeServiceEquipmentRepository(equipment ...serviceequipment.ServiceEquipment) *fakeServiceEquipmentRepository {
	f := &fakeServiceEquipmentRepository{byID: make(map[uuid.UUID]serviceequipment.ServiceEquipment)}
	for _, e := range equipment {
		f.byID[e.ID] = e
	}
	return f
}

func (f *fakeServiceEquipmentRepository) Get(_ context.Context, id uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	e, ok := f.byID[id]
	if !ok {
		return serviceequipment.ServiceEquipment{}, apperror.NotFound("service equipment not found")
	}
	return e, nil
}

func (f *fakeServiceEquipmentRepository) List(_ context.Context) ([]serviceequipment.ServiceEquipment, error) {
	equipment := make([]serviceequipment.ServiceEquipment, 0, len(f.byID))
	for _, e := range f.byID {
		equipment = append(equipment, e)
	}
	return equipment, nil
}

func (f *fakeServiceEquipmentRepository) Create(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	f.createCalled = true
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeServiceEquipmentRepository) Update(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	f.updateCalled = true
	if _, ok := f.byID[e.ID]; !ok {
		return serviceequipment.ServiceEquipment{}, apperror.NotFound("service equipment not found")
	}
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeServiceEquipmentRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("service equipment not found")
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeServiceEquipmentRepository) GetActiveByDeviceID(_ context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	for _, e := range f.byID {
		if e.DeviceID == deviceID && e.Active() {
			return e, nil
		}
	}
	return serviceequipment.ServiceEquipment{}, apperror.NotFound("no active service equipment assignment for device")
}

func (f *fakeServiceEquipmentRepository) GetLatestByDeviceID(_ context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	var latest serviceequipment.ServiceEquipment
	found := false
	for _, e := range f.byID {
		if e.DeviceID == deviceID && (!found || e.CreatedAt.After(latest.CreatedAt)) {
			latest = e
			found = true
		}
	}
	if !found {
		return serviceequipment.ServiceEquipment{}, apperror.NotFound("no service equipment assignment for device")
	}
	return latest, nil
}

func (f *fakeServiceEquipmentRepository) ListActiveByServiceID(_ context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error) {
	var equipment []serviceequipment.ServiceEquipment
	for _, e := range f.byID {
		if e.ServiceID == serviceID && e.Active() {
			equipment = append(equipment, e)
		}
	}
	return equipment, nil
}

var _ serviceequipment.ServiceEquipmentRepository = (*fakeServiceEquipmentRepository)(nil)

func validServiceEquipment() serviceequipment.ServiceEquipment {
	return serviceequipment.ServiceEquipment{
		ServiceID: uuid.New(),
		DeviceID:  uuid.New(),
		Role:      serviceequipment.EquipmentRoleONU,
	}
}

func TestServiceEquipmentServiceCreateSucceeds(t *testing.T) {
	repo := newFakeServiceEquipmentRepository()
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	created, err := svc.Create(context.Background(), validServiceEquipment())
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

func TestServiceEquipmentServiceCreateRejectsInvalidServiceEquipmentWithoutPersisting(t *testing.T) {
	repo := newFakeServiceEquipmentRepository()
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	_, err := svc.Create(context.Background(), serviceequipment.ServiceEquipment{}) // no ServiceID, DeviceID, Role

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

// TestServiceEquipmentServiceCreateRejectsSecondActiveAssignmentForSameDevice
// is goal 2's central proof: "attempting to create another active
// assignment for the same DeviceID must return a Conflict error."
func TestServiceEquipmentServiceCreateRejectsSecondActiveAssignmentForSameDevice(t *testing.T) {
	deviceID := uuid.New()
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	second := validServiceEquipment()
	second.DeviceID = deviceID // same device, still active (no RemovedAt)

	_, err := svc.Create(context.Background(), second)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite an existing active assignment; the uniqueness check must run first")
	}
}

// TestServiceEquipmentServiceCreateAllowsHistoricalReassignment is goal
// 2's other central proof: "historical assignments remain allowed" — once
// the existing assignment for a device has been removed (RemovedAt set),
// a new active assignment for that same device must succeed.
func TestServiceEquipmentServiceCreateAllowsHistoricalReassignment(t *testing.T) {
	deviceID := uuid.New()
	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	historical := validServiceEquipment()
	historical.ID = uuid.New()
	historical.DeviceID = deviceID
	historical.RemovedAt = &removedAt // no longer active
	repo := newFakeServiceEquipmentRepository(historical)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	replacement := validServiceEquipment()
	replacement.DeviceID = deviceID // same device, but the old assignment is history

	created, err := svc.Create(context.Background(), replacement)
	if err != nil {
		t.Fatalf("Create() = %v, want success since the existing assignment for this device is historical", err)
	}
	if !created.Active() {
		t.Error("created assignment is not active, want an active new assignment")
	}
	if !repo.createCalled {
		t.Error("repository Create() was never called")
	}
}

// TestServiceEquipmentServiceCreateAllowsCreatingAlreadyHistoricalRecord
// proves creating a record that is historical from the start (RemovedAt
// already set on Create) never triggers the uniqueness check at all, even
// when the device currently has a real active assignment — a record that
// is not active by definition cannot violate "only one active assignment
// per device."
func TestServiceEquipmentServiceCreateAllowsCreatingAlreadyHistoricalRecord(t *testing.T) {
	deviceID := uuid.New()
	active := validServiceEquipment()
	active.ID = uuid.New()
	active.DeviceID = deviceID
	repo := newFakeServiceEquipmentRepository(active)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	removedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	backfilled := validServiceEquipment()
	backfilled.DeviceID = deviceID
	backfilled.RemovedAt = &removedAt // historical from the moment it is created

	if _, err := svc.Create(context.Background(), backfilled); err != nil {
		t.Fatalf("Create() = %v, want success for a record that is historical from creation", err)
	}
}

func TestServiceEquipmentServiceUpdateSucceeds(t *testing.T) {
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

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

// TestServiceEquipmentServiceUpdateAllowsUpdatingTheActiveRecordItself
// proves the uniqueness check excludes the record being updated: editing
// a field on an already-active assignment must not conflict with itself.
func TestServiceEquipmentServiceUpdateAllowsUpdatingTheActiveRecordItself(t *testing.T) {
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	toUpdate := existing
	toUpdate.Role = serviceequipment.EquipmentRoleRouter // still active, same DeviceID, same ID

	if _, err := svc.Update(context.Background(), toUpdate); err != nil {
		t.Fatalf("Update() = %v, want success updating the active record itself", err)
	}
}

// TestServiceEquipmentServiceUpdateRejectsReassigningDeviceWithExistingActiveAssignment
// proves reassigning DeviceID on Update is subject to the same uniqueness
// rule as Create: the target device must not already have a different
// active assignment.
func TestServiceEquipmentServiceUpdateRejectsReassigningDeviceWithExistingActiveAssignment(t *testing.T) {
	busyDeviceID := uuid.New()
	busy := validServiceEquipment()
	busy.ID = uuid.New()
	busy.DeviceID = busyDeviceID

	toReassign := validServiceEquipment()
	toReassign.ID = uuid.New()
	repo := newFakeServiceEquipmentRepository(busy, toReassign)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	toReassign.DeviceID = busyDeviceID // now targets a device that's already actively assigned

	_, err := svc.Update(context.Background(), toReassign)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestServiceEquipmentServiceUpdateRejectsInvalidServiceEquipmentWithoutPersisting(t *testing.T) {
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	invalid := existing
	invalid.Role = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestServiceEquipmentServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeServiceEquipmentRepository()
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceEquipmentServiceListDelegatesToRepository(t *testing.T) {
	a := validServiceEquipment()
	a.ID = uuid.New()
	b := validServiceEquipment()
	b.ID = uuid.New()
	repo := newFakeServiceEquipmentRepository(a, b)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	equipment, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(equipment) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(equipment))
	}
}

func TestServiceEquipmentServiceDeleteSucceeds(t *testing.T) {
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

// TestServiceEquipmentServiceDeleteMarksDeviceUnused proves Delete's own
// Device-status side effect: ServiceDetailView.vue's "Remove Equipment"
// button hard-deletes the ServiceEquipment row entirely (see
// internal/serviceequipment/httpapi's DELETE route) rather than going
// through Update's RemovedAt-transition path, so Delete needs the
// identical side effect Update carries -- deleting an active record must
// still push its Device back to Unused.
func TestServiceEquipmentServiceDeleteMarksDeviceUnused(t *testing.T) {
	deviceID := uuid.New()
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore(inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusActive})
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}
	if len(devices.updateCalls) != 1 {
		t.Fatalf("device Update calls = %d, want 1", len(devices.updateCalls))
	}
	if devices.updateCalls[0].Status != inventory.DeviceStatusUnused {
		t.Errorf("device Status = %q, want %q", devices.updateCalls[0].Status, inventory.DeviceStatusUnused)
	}
}

// TestServiceEquipmentServiceDeleteOfAlreadyRemovedRecordDoesNotTouchDevice
// proves the Unused side effect only fires when the deleted record was
// itself still active -- deleting old, already-removed history must not
// touch a Device that may since have been legitimately reattached.
func TestServiceEquipmentServiceDeleteOfAlreadyRemovedRecordDoesNotTouchDevice(t *testing.T) {
	deviceID := uuid.New()
	removedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	existing.RemovedAt = &removedAt
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore(inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusActive})
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}
	if len(devices.updateCalls) != 0 {
		t.Errorf("device Update calls = %d, want 0 (record was already removed)", len(devices.updateCalls))
	}
}

func TestServiceEquipmentServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeServiceEquipmentRepository()
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

// TestServiceEquipmentServiceCreateMarksDeviceActive proves Create's
// Device-status side effect: a newly created active assignment flips its
// Device from Unused (the fakeDeviceStore default) to Active.
func TestServiceEquipmentServiceCreateMarksDeviceActive(t *testing.T) {
	deviceID := uuid.New()
	repo := newFakeServiceEquipmentRepository()
	devices := newFakeDeviceStore(inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusUnused})
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	e := validServiceEquipment()
	e.DeviceID = deviceID

	if _, err := svc.Create(context.Background(), e); err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if len(devices.updateCalls) != 1 {
		t.Fatalf("device Update calls = %d, want 1", len(devices.updateCalls))
	}
	if devices.updateCalls[0].Status != inventory.DeviceStatusActive {
		t.Errorf("device Status = %q, want %q", devices.updateCalls[0].Status, inventory.DeviceStatusActive)
	}
}

// TestServiceEquipmentServiceCreateOfHistoricalRecordDoesNotTouchDevice
// proves the Active side effect only fires for a record that is itself
// active on creation — see Create's own doc comment.
func TestServiceEquipmentServiceCreateOfHistoricalRecordDoesNotTouchDevice(t *testing.T) {
	deviceID := uuid.New()
	repo := newFakeServiceEquipmentRepository()
	devices := newFakeDeviceStore()
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	removedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	e := validServiceEquipment()
	e.DeviceID = deviceID
	e.RemovedAt = &removedAt

	if _, err := svc.Create(context.Background(), e); err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if len(devices.updateCalls) != 0 {
		t.Errorf("device Update calls = %d, want 0 (record was never active)", len(devices.updateCalls))
	}
}

// TestServiceEquipmentServiceUpdateMarksDeviceUnusedOnRemoval proves
// Update's Device-status side effect: a write that transitions a
// previously-active record to removed (RemovedAt going from nil to set)
// flips its Device back to Unused.
func TestServiceEquipmentServiceUpdateMarksDeviceUnusedOnRemoval(t *testing.T) {
	deviceID := uuid.New()
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore(inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusActive})
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	now := time.Now()
	toRemove := existing
	toRemove.RemovedAt = &now

	if _, err := svc.Update(context.Background(), toRemove); err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if len(devices.updateCalls) != 1 {
		t.Fatalf("device Update calls = %d, want 1", len(devices.updateCalls))
	}
	if devices.updateCalls[0].Status != inventory.DeviceStatusUnused {
		t.Errorf("device Status = %q, want %q", devices.updateCalls[0].Status, inventory.DeviceStatusUnused)
	}
}

// TestServiceEquipmentServiceUpdateOfAlreadyRemovedRecordDoesNotTouchDevice
// proves the Unused side effect only fires on the active-to-removed
// transition itself, not on every subsequent edit to an already-removed
// record — see Update's own doc comment on why: the Device may since
// have been legitimately reattached (a new, different active record) and
// must not be silently pulled back to Unused by an unrelated edit to old
// history.
func TestServiceEquipmentServiceUpdateOfAlreadyRemovedRecordDoesNotTouchDevice(t *testing.T) {
	deviceID := uuid.New()
	removedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	existing.RemovedAt = &removedAt
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore(inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusActive})
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	toUpdate := existing
	toUpdate.Description = "editing history, not removing anything new"

	if _, err := svc.Update(context.Background(), toUpdate); err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if len(devices.updateCalls) != 0 {
		t.Errorf("device Update calls = %d, want 0 (record was already removed before this edit)", len(devices.updateCalls))
	}
}

// TestServiceEquipmentServiceMarkDeviceUnusedNeverUnretires proves
// markDeviceUnused's own idempotency guard: a Retired Device (fully
// deauthorized from its OLT) must never be silently pulled back to
// Unused just because an old ServiceEquipment record for it is also
// being marked removed here.
func TestServiceEquipmentServiceMarkDeviceUnusedNeverUnretires(t *testing.T) {
	deviceID := uuid.New()
	existing := validServiceEquipment()
	existing.ID = uuid.New()
	existing.DeviceID = deviceID
	repo := newFakeServiceEquipmentRepository(existing)
	devices := newFakeDeviceStore(inventory.Device{Metadata: inventory.Metadata{ID: deviceID}, Status: inventory.DeviceStatusRetired})
	svc := service.NewServiceEquipmentService(repo, devices, devices)

	now := time.Now()
	toRemove := existing
	toRemove.RemovedAt = &now

	if _, err := svc.Update(context.Background(), toRemove); err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if len(devices.updateCalls) != 0 {
		t.Errorf("device Update calls = %d, want 0 (a Retired device must never be un-retired)", len(devices.updateCalls))
	}
}
