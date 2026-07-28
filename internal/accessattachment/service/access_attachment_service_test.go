package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAccessAttachmentRepository is an in-memory
// accessattachment.AccessAttachmentRepository. Like
// internal/serviceequipment/service's fakeServiceEquipmentRepository, it
// exists so AccessAttachmentService's business logic — validate, enforce
// active-attachment uniqueness, then delegate — is tested without a real
// database; the layer below (this domain's postgres package) already
// covers GetActiveByServiceEquipmentID's own SQL against real
// PostgreSQL. GetActiveByServiceEquipmentID here does the same linear
// scan a real query's WHERE service_equipment_id = $1 AND removed_at IS
// NULL would express declaratively, which is exactly what makes it a
// faithful enough fake for testing the service layer's use of it.
type fakeAccessAttachmentRepository struct {
	byID         map[uuid.UUID]accessattachment.AccessAttachment
	createCalled bool
	updateCalled bool
}

func newFakeAccessAttachmentRepository(attachments ...accessattachment.AccessAttachment) *fakeAccessAttachmentRepository {
	f := &fakeAccessAttachmentRepository{byID: make(map[uuid.UUID]accessattachment.AccessAttachment)}
	for _, a := range attachments {
		f.byID[a.ID] = a
	}
	return f
}

func (f *fakeAccessAttachmentRepository) Get(_ context.Context, id uuid.UUID) (accessattachment.AccessAttachment, error) {
	a, ok := f.byID[id]
	if !ok {
		return accessattachment.AccessAttachment{}, apperror.NotFound("access attachment not found")
	}
	return a, nil
}

func (f *fakeAccessAttachmentRepository) List(_ context.Context) ([]accessattachment.AccessAttachment, error) {
	attachments := make([]accessattachment.AccessAttachment, 0, len(f.byID))
	for _, a := range f.byID {
		attachments = append(attachments, a)
	}
	return attachments, nil
}

func (f *fakeAccessAttachmentRepository) Create(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	f.createCalled = true
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAccessAttachmentRepository) Update(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	f.updateCalled = true
	if _, ok := f.byID[a.ID]; !ok {
		return accessattachment.AccessAttachment{}, apperror.NotFound("access attachment not found")
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAccessAttachmentRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("access attachment not found")
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeAccessAttachmentRepository) GetActiveByServiceEquipmentID(_ context.Context, serviceEquipmentID uuid.UUID) (accessattachment.AccessAttachment, error) {
	for _, a := range f.byID {
		if a.ServiceEquipmentID == serviceEquipmentID && a.Active() {
			return a, nil
		}
	}
	return accessattachment.AccessAttachment{}, apperror.NotFound("no active access attachment for service equipment")
}

var _ accessattachment.AccessAttachmentRepository = (*fakeAccessAttachmentRepository)(nil)

func validAccessAttachment() accessattachment.AccessAttachment {
	return accessattachment.AccessAttachment{
		AccessInterfaceID:  uuid.New(),
		ServiceEquipmentID: uuid.New(),
	}
}

func TestAccessAttachmentServiceCreateSucceeds(t *testing.T) {
	repo := newFakeAccessAttachmentRepository()
	svc := service.NewAccessAttachmentService(repo)

	created, err := svc.Create(context.Background(), validAccessAttachment())
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

func TestAccessAttachmentServiceCreateRejectsInvalidAccessAttachmentWithoutPersisting(t *testing.T) {
	repo := newFakeAccessAttachmentRepository()
	svc := service.NewAccessAttachmentService(repo)

	_, err := svc.Create(context.Background(), accessattachment.AccessAttachment{}) // no AccessInterfaceID, ServiceEquipmentID

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

// TestAccessAttachmentServiceCreateRejectsSecondActiveAttachmentForSameServiceEquipment
// is this milestone's goal 2 central proof: "attempting to create another
// active attachment for the same ServiceEquipmentID must return a
// Conflict error."
func TestAccessAttachmentServiceCreateRejectsSecondActiveAttachmentForSameServiceEquipment(t *testing.T) {
	serviceEquipmentID := uuid.New()
	existing := validAccessAttachment()
	existing.ID = uuid.New()
	existing.ServiceEquipmentID = serviceEquipmentID
	repo := newFakeAccessAttachmentRepository(existing)
	svc := service.NewAccessAttachmentService(repo)

	second := validAccessAttachment()
	second.ServiceEquipmentID = serviceEquipmentID // same equipment, still active (no RemovedAt)

	_, err := svc.Create(context.Background(), second)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite an existing active attachment; the uniqueness check must run first")
	}
}

// TestAccessAttachmentServiceCreateAllowsHistoricalReassignment is goal
// 2's other central proof: "historical moves are allowed" — once the
// existing attachment for equipment has been removed (RemovedAt set), a
// new active attachment for that same equipment must succeed.
func TestAccessAttachmentServiceCreateAllowsHistoricalReassignment(t *testing.T) {
	serviceEquipmentID := uuid.New()
	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	historical := validAccessAttachment()
	historical.ID = uuid.New()
	historical.ServiceEquipmentID = serviceEquipmentID
	historical.RemovedAt = &removedAt // no longer active
	repo := newFakeAccessAttachmentRepository(historical)
	svc := service.NewAccessAttachmentService(repo)

	replacement := validAccessAttachment()
	replacement.ServiceEquipmentID = serviceEquipmentID // same equipment, but the old attachment is history

	created, err := svc.Create(context.Background(), replacement)
	if err != nil {
		t.Fatalf("Create() = %v, want success since the existing attachment for this equipment is historical", err)
	}
	if !created.Active() {
		t.Error("created attachment is not active, want an active new attachment")
	}
	if !repo.createCalled {
		t.Error("repository Create() was never called")
	}
}

// TestAccessAttachmentServiceCreateAllowsCreatingAlreadyHistoricalRecord
// proves creating a record that is historical from the start (RemovedAt
// already set on Create) never triggers the uniqueness check at all,
// even when the equipment currently has a real active attachment — a
// record that is not active by definition cannot violate "only one
// active attachment per ServiceEquipment."
func TestAccessAttachmentServiceCreateAllowsCreatingAlreadyHistoricalRecord(t *testing.T) {
	serviceEquipmentID := uuid.New()
	active := validAccessAttachment()
	active.ID = uuid.New()
	active.ServiceEquipmentID = serviceEquipmentID
	repo := newFakeAccessAttachmentRepository(active)
	svc := service.NewAccessAttachmentService(repo)

	removedAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	backfilled := validAccessAttachment()
	backfilled.ServiceEquipmentID = serviceEquipmentID
	backfilled.RemovedAt = &removedAt // historical from the moment it is created

	if _, err := svc.Create(context.Background(), backfilled); err != nil {
		t.Fatalf("Create() = %v, want success for a record that is historical from creation", err)
	}
}

func TestAccessAttachmentServiceUpdateSucceeds(t *testing.T) {
	existing := validAccessAttachment()
	existing.ID = uuid.New()
	repo := newFakeAccessAttachmentRepository(existing)
	svc := service.NewAccessAttachmentService(repo)

	toUpdate := existing
	toUpdate.RemovalReason = "Customer moved"

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.RemovalReason != "Customer moved" {
		t.Errorf("RemovalReason = %q, want %q", updated.RemovalReason, "Customer moved")
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

// TestAccessAttachmentServiceUpdateAllowsUpdatingTheActiveRecordItself
// proves the uniqueness check excludes the record being updated: editing
// a field on an already-active attachment must not conflict with itself.
func TestAccessAttachmentServiceUpdateAllowsUpdatingTheActiveRecordItself(t *testing.T) {
	existing := validAccessAttachment()
	existing.ID = uuid.New()
	repo := newFakeAccessAttachmentRepository(existing)
	svc := service.NewAccessAttachmentService(repo)

	toUpdate := existing
	toUpdate.RemovalReason = "still active, same equipment, same ID"

	if _, err := svc.Update(context.Background(), toUpdate); err != nil {
		t.Fatalf("Update() = %v, want success updating the active record itself", err)
	}
}

// TestAccessAttachmentServiceUpdateRejectsReassigningServiceEquipmentWithExistingActiveAttachment
// proves reassigning ServiceEquipmentID on Update is subject to the same
// uniqueness rule as Create: the target equipment must not already have
// a different active attachment.
func TestAccessAttachmentServiceUpdateRejectsReassigningServiceEquipmentWithExistingActiveAttachment(t *testing.T) {
	busyServiceEquipmentID := uuid.New()
	busy := validAccessAttachment()
	busy.ID = uuid.New()
	busy.ServiceEquipmentID = busyServiceEquipmentID

	toReassign := validAccessAttachment()
	toReassign.ID = uuid.New()
	repo := newFakeAccessAttachmentRepository(busy, toReassign)
	svc := service.NewAccessAttachmentService(repo)

	toReassign.ServiceEquipmentID = busyServiceEquipmentID // now targets equipment that's already actively attached

	_, err := svc.Update(context.Background(), toReassign)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestAccessAttachmentServiceUpdateRejectsInvalidAccessAttachmentWithoutPersisting(t *testing.T) {
	existing := validAccessAttachment()
	existing.ID = uuid.New()
	repo := newFakeAccessAttachmentRepository(existing)
	svc := service.NewAccessAttachmentService(repo)

	invalid := existing
	invalid.ServiceEquipmentID = uuid.Nil // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestAccessAttachmentServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeAccessAttachmentRepository()
	svc := service.NewAccessAttachmentService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAccessAttachmentServiceListDelegatesToRepository(t *testing.T) {
	a := validAccessAttachment()
	a.ID = uuid.New()
	b := validAccessAttachment()
	b.ID = uuid.New()
	repo := newFakeAccessAttachmentRepository(a, b)
	svc := service.NewAccessAttachmentService(repo)

	attachments, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(attachments) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(attachments))
	}
}

func TestAccessAttachmentServiceDeleteSucceeds(t *testing.T) {
	existing := validAccessAttachment()
	existing.ID = uuid.New()
	repo := newFakeAccessAttachmentRepository(existing)
	svc := service.NewAccessAttachmentService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAccessAttachmentServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeAccessAttachmentRepository()
	svc := service.NewAccessAttachmentService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
