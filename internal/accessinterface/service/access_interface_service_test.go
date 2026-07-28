package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAccessInterfaceRepository is an in-memory
// accessinterface.AccessInterfaceRepository. Like
// internal/olt/service/olt_service_test.go's fakeOLTRepository, it
// exists so AccessInterfaceService's business logic — validate, then
// delegate — is tested without a real database;
// internal/accessinterface/postgres/access_interface_test.go already
// covers the repository itself against real PostgreSQL. It tracks
// whether Create/Update were actually invoked, which is what lets
// TestAccessInterfaceServiceCreateRejectsInvalidAccessInterfaceWithoutPersisting
// prove validation happens before any repository call.
type fakeAccessInterfaceRepository struct {
	byID         map[uuid.UUID]accessinterface.AccessInterface
	createCalled bool
	updateCalled bool
}

func newFakeAccessInterfaceRepository(interfaces ...accessinterface.AccessInterface) *fakeAccessInterfaceRepository {
	f := &fakeAccessInterfaceRepository{byID: make(map[uuid.UUID]accessinterface.AccessInterface)}
	for _, a := range interfaces {
		f.byID[a.ID] = a
	}
	return f
}

func (f *fakeAccessInterfaceRepository) Get(_ context.Context, id uuid.UUID) (accessinterface.AccessInterface, error) {
	a, ok := f.byID[id]
	if !ok {
		return accessinterface.AccessInterface{}, apperror.NotFound("access interface not found")
	}
	return a, nil
}

func (f *fakeAccessInterfaceRepository) List(_ context.Context) ([]accessinterface.AccessInterface, error) {
	interfaces := make([]accessinterface.AccessInterface, 0, len(f.byID))
	for _, a := range f.byID {
		interfaces = append(interfaces, a)
	}
	return interfaces, nil
}

func (f *fakeAccessInterfaceRepository) Create(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	f.createCalled = true
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAccessInterfaceRepository) Update(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	f.updateCalled = true
	if _, ok := f.byID[a.ID]; !ok {
		return accessinterface.AccessInterface{}, apperror.NotFound("access interface not found")
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAccessInterfaceRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("access interface not found")
	}
	delete(f.byID, id)
	return nil
}

var _ accessinterface.AccessInterfaceRepository = (*fakeAccessInterfaceRepository)(nil)

func validAccessInterface() accessinterface.AccessInterface {
	return accessinterface.AccessInterface{
		PONPortID:  uuid.New(),
		Technology: accessinterface.TechnologyGPON,
		Name:       "gpon-0/1/1",
		Status:     accessinterface.StatusActive,
	}
}

func TestAccessInterfaceServiceCreateSucceeds(t *testing.T) {
	repo := newFakeAccessInterfaceRepository()
	svc := service.NewAccessInterfaceService(repo)

	created, err := svc.Create(context.Background(), validAccessInterface())
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

func TestAccessInterfaceServiceCreateRejectsInvalidAccessInterfaceWithoutPersisting(t *testing.T) {
	repo := newFakeAccessInterfaceRepository()
	svc := service.NewAccessInterfaceService(repo)

	_, err := svc.Create(context.Background(), accessinterface.AccessInterface{}) // no PONPortID, Technology, Name, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestAccessInterfaceServiceUpdateSucceeds(t *testing.T) {
	existing := validAccessInterface()
	existing.ID = uuid.New()
	repo := newFakeAccessInterfaceRepository(existing)
	svc := service.NewAccessInterfaceService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = accessinterface.StatusDisabled

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != accessinterface.StatusDisabled {
		t.Errorf("Status = %q, want %q", updated.Status, accessinterface.StatusDisabled)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestAccessInterfaceServiceUpdateRejectsInvalidAccessInterfaceWithoutPersisting(t *testing.T) {
	existing := validAccessInterface()
	existing.ID = uuid.New()
	repo := newFakeAccessInterfaceRepository(existing)
	svc := service.NewAccessInterfaceService(repo)

	invalid := existing
	invalid.Name = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestAccessInterfaceServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeAccessInterfaceRepository()
	svc := service.NewAccessInterfaceService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAccessInterfaceServiceListDelegatesToRepository(t *testing.T) {
	a := validAccessInterface()
	a.ID = uuid.New()
	b := validAccessInterface()
	b.ID = uuid.New()
	repo := newFakeAccessInterfaceRepository(a, b)
	svc := service.NewAccessInterfaceService(repo)

	interfaces, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(interfaces) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(interfaces))
	}
}

func TestAccessInterfaceServiceDeleteSucceeds(t *testing.T) {
	existing := validAccessInterface()
	existing.ID = uuid.New()
	repo := newFakeAccessInterfaceRepository(existing)
	svc := service.NewAccessInterfaceService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAccessInterfaceServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeAccessInterfaceRepository()
	svc := service.NewAccessInterfaceService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
