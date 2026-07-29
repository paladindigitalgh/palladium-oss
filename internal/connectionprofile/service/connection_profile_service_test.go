package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeConnectionProfileRepository is an in-memory
// connectionprofile.ConnectionProfileRepository. Like
// internal/catalog/service/catalog_service_test.go's
// fakeCatalogRepository, it exists so ConnectionProfileService's
// business logic — validate, then delegate — is tested without a real
// database; internal/connectionprofile/postgres's own tests already
// cover the repository itself against real PostgreSQL. It tracks
// whether Create/Update were actually invoked, which is what lets
// TestConnectionProfileServiceCreateRejectsInvalidProfileWithoutPersisting
// prove validation happens before any repository call.
type fakeConnectionProfileRepository struct {
	byID         map[uuid.UUID]connectionprofile.ConnectionProfile
	createCalled bool
	updateCalled bool
}

func newFakeConnectionProfileRepository(profiles ...connectionprofile.ConnectionProfile) *fakeConnectionProfileRepository {
	f := &fakeConnectionProfileRepository{byID: make(map[uuid.UUID]connectionprofile.ConnectionProfile)}
	for _, p := range profiles {
		f.byID[p.ID] = p
	}
	return f
}

func (f *fakeConnectionProfileRepository) Get(_ context.Context, id uuid.UUID) (connectionprofile.ConnectionProfile, error) {
	p, ok := f.byID[id]
	if !ok {
		return connectionprofile.ConnectionProfile{}, apperror.NotFound("connection profile not found")
	}
	return p, nil
}

func (f *fakeConnectionProfileRepository) List(_ context.Context) ([]connectionprofile.ConnectionProfile, error) {
	profiles := make([]connectionprofile.ConnectionProfile, 0, len(f.byID))
	for _, p := range f.byID {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (f *fakeConnectionProfileRepository) Create(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	f.createCalled = true
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeConnectionProfileRepository) Update(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	f.updateCalled = true
	if _, ok := f.byID[p.ID]; !ok {
		return connectionprofile.ConnectionProfile{}, apperror.NotFound("connection profile not found")
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeConnectionProfileRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("connection profile not found")
	}
	delete(f.byID, id)
	return nil
}

var _ connectionprofile.ConnectionProfileRepository = (*fakeConnectionProfileRepository)(nil)

func validConnectionProfile() connectionprofile.ConnectionProfile {
	return connectionprofile.ConnectionProfile{
		Name:          "Standard SSH",
		HostKeyPolicy: connectionprofile.HostKeyPolicyStrict,
	}
}

func TestConnectionProfileServiceCreateSucceeds(t *testing.T) {
	repo := newFakeConnectionProfileRepository()
	svc := service.NewConnectionProfileService(repo)

	created, err := svc.Create(context.Background(), validConnectionProfile())
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

func TestConnectionProfileServiceCreateRejectsInvalidProfileWithoutPersisting(t *testing.T) {
	repo := newFakeConnectionProfileRepository()
	svc := service.NewConnectionProfileService(repo)

	_, err := svc.Create(context.Background(), connectionprofile.ConnectionProfile{}) // no Name, HostKeyPolicy

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestConnectionProfileServiceUpdateSucceeds(t *testing.T) {
	existing := validConnectionProfile()
	existing.ID = uuid.New()
	repo := newFakeConnectionProfileRepository(existing)
	svc := service.NewConnectionProfileService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.HostKeyPolicy = connectionprofile.HostKeyPolicyInsecure

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.HostKeyPolicy != connectionprofile.HostKeyPolicyInsecure {
		t.Errorf("HostKeyPolicy = %q, want %q", updated.HostKeyPolicy, connectionprofile.HostKeyPolicyInsecure)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestConnectionProfileServiceUpdateRejectsInvalidProfileWithoutPersisting(t *testing.T) {
	existing := validConnectionProfile()
	existing.ID = uuid.New()
	repo := newFakeConnectionProfileRepository(existing)
	svc := service.NewConnectionProfileService(repo)

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

func TestConnectionProfileServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeConnectionProfileRepository()
	svc := service.NewConnectionProfileService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestConnectionProfileServiceListDelegatesToRepository(t *testing.T) {
	a := validConnectionProfile()
	a.ID = uuid.New()
	b := validConnectionProfile()
	b.ID = uuid.New()
	repo := newFakeConnectionProfileRepository(a, b)
	svc := service.NewConnectionProfileService(repo)

	profiles, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(profiles))
	}
}

func TestConnectionProfileServiceDeleteSucceeds(t *testing.T) {
	existing := validConnectionProfile()
	existing.ID = uuid.New()
	repo := newFakeConnectionProfileRepository(existing)
	svc := service.NewConnectionProfileService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestConnectionProfileServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeConnectionProfileRepository()
	svc := service.NewConnectionProfileService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
