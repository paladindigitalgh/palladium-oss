package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAccessNetworkRepository is an in-memory
// accessnetwork.AccessNetworkRepository. Like
// internal/catalog/service/catalog_service_test.go's
// fakeCatalogRepository, it exists so AccessNetworkService's business
// logic — validate, then delegate — is tested without a real database;
// internal/accessnetwork/postgres/access_network_test.go already covers
// the repository itself against real PostgreSQL. It tracks whether
// Create/Update were actually invoked, which is what lets
// TestAccessNetworkServiceCreateRejectsInvalidAccessNetworkWithoutPersisting
// prove validation happens before any repository call.
type fakeAccessNetworkRepository struct {
	byID         map[uuid.UUID]accessnetwork.AccessNetwork
	createCalled bool
	updateCalled bool
}

func newFakeAccessNetworkRepository(networks ...accessnetwork.AccessNetwork) *fakeAccessNetworkRepository {
	f := &fakeAccessNetworkRepository{byID: make(map[uuid.UUID]accessnetwork.AccessNetwork)}
	for _, a := range networks {
		f.byID[a.ID] = a
	}
	return f
}

func (f *fakeAccessNetworkRepository) Get(_ context.Context, id uuid.UUID) (accessnetwork.AccessNetwork, error) {
	a, ok := f.byID[id]
	if !ok {
		return accessnetwork.AccessNetwork{}, apperror.NotFound("access network not found")
	}
	return a, nil
}

func (f *fakeAccessNetworkRepository) List(_ context.Context) ([]accessnetwork.AccessNetwork, error) {
	networks := make([]accessnetwork.AccessNetwork, 0, len(f.byID))
	for _, a := range f.byID {
		networks = append(networks, a)
	}
	return networks, nil
}

func (f *fakeAccessNetworkRepository) Create(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	f.createCalled = true
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAccessNetworkRepository) Update(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	f.updateCalled = true
	if _, ok := f.byID[a.ID]; !ok {
		return accessnetwork.AccessNetwork{}, apperror.NotFound("access network not found")
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAccessNetworkRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("access network not found")
	}
	delete(f.byID, id)
	return nil
}

var _ accessnetwork.AccessNetworkRepository = (*fakeAccessNetworkRepository)(nil)

func validAccessNetwork() accessnetwork.AccessNetwork {
	return accessnetwork.AccessNetwork{
		Name:   "North Region GPON",
		Status: accessnetwork.AccessNetworkStatusActive,
	}
}

func TestAccessNetworkServiceCreateSucceeds(t *testing.T) {
	repo := newFakeAccessNetworkRepository()
	svc := service.NewAccessNetworkService(repo)

	created, err := svc.Create(context.Background(), validAccessNetwork())
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

func TestAccessNetworkServiceCreateRejectsInvalidAccessNetworkWithoutPersisting(t *testing.T) {
	repo := newFakeAccessNetworkRepository()
	svc := service.NewAccessNetworkService(repo)

	_, err := svc.Create(context.Background(), accessnetwork.AccessNetwork{}) // no Name, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestAccessNetworkServiceUpdateSucceeds(t *testing.T) {
	existing := validAccessNetwork()
	existing.ID = uuid.New()
	repo := newFakeAccessNetworkRepository(existing)
	svc := service.NewAccessNetworkService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = accessnetwork.AccessNetworkStatusInactive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != accessnetwork.AccessNetworkStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, accessnetwork.AccessNetworkStatusInactive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestAccessNetworkServiceUpdateRejectsInvalidAccessNetworkWithoutPersisting(t *testing.T) {
	existing := validAccessNetwork()
	existing.ID = uuid.New()
	repo := newFakeAccessNetworkRepository(existing)
	svc := service.NewAccessNetworkService(repo)

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

func TestAccessNetworkServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeAccessNetworkRepository()
	svc := service.NewAccessNetworkService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAccessNetworkServiceListDelegatesToRepository(t *testing.T) {
	a := validAccessNetwork()
	a.ID = uuid.New()
	b := validAccessNetwork()
	b.ID = uuid.New()
	repo := newFakeAccessNetworkRepository(a, b)
	svc := service.NewAccessNetworkService(repo)

	networks, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(networks) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(networks))
	}
}

func TestAccessNetworkServiceDeleteSucceeds(t *testing.T) {
	existing := validAccessNetwork()
	existing.ID = uuid.New()
	repo := newFakeAccessNetworkRepository(existing)
	svc := service.NewAccessNetworkService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAccessNetworkServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeAccessNetworkRepository()
	svc := service.NewAccessNetworkService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
