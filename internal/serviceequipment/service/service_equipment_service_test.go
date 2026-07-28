package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/service"
)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

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
	svc := service.NewServiceEquipmentService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceEquipmentServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeServiceEquipmentRepository()
	svc := service.NewServiceEquipmentService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
