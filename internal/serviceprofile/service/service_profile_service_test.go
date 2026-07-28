package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/service"
)

// fakeServiceProfileRepository is an in-memory
// serviceprofile.ServiceProfileRepository. Like
// internal/catalog/service/catalog_service_test.go's
// fakeCatalogRepository, it exists so ServiceProfileService's business
// logic — validate, then delegate — is tested without a real database;
// internal/serviceprofile/postgres/service_profile_test.go already
// covers the repository itself against real PostgreSQL. It tracks
// whether Create/Update were actually invoked, which is what lets
// TestServiceProfileServiceCreateRejectsInvalidServiceProfileWithoutPersisting
// prove validation happens before any repository call.
type fakeServiceProfileRepository struct {
	byID         map[uuid.UUID]serviceprofile.ServiceProfile
	createCalled bool
	updateCalled bool
}

func newFakeServiceProfileRepository(profiles ...serviceprofile.ServiceProfile) *fakeServiceProfileRepository {
	f := &fakeServiceProfileRepository{byID: make(map[uuid.UUID]serviceprofile.ServiceProfile)}
	for _, p := range profiles {
		f.byID[p.ID] = p
	}
	return f
}

func (f *fakeServiceProfileRepository) Get(_ context.Context, id uuid.UUID) (serviceprofile.ServiceProfile, error) {
	p, ok := f.byID[id]
	if !ok {
		return serviceprofile.ServiceProfile{}, apperror.NotFound("service profile not found")
	}
	return p, nil
}

func (f *fakeServiceProfileRepository) List(_ context.Context) ([]serviceprofile.ServiceProfile, error) {
	profiles := make([]serviceprofile.ServiceProfile, 0, len(f.byID))
	for _, p := range f.byID {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (f *fakeServiceProfileRepository) Create(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	f.createCalled = true
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeServiceProfileRepository) Update(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	f.updateCalled = true
	if _, ok := f.byID[p.ID]; !ok {
		return serviceprofile.ServiceProfile{}, apperror.NotFound("service profile not found")
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeServiceProfileRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("service profile not found")
	}
	delete(f.byID, id)
	return nil
}

var _ serviceprofile.ServiceProfileRepository = (*fakeServiceProfileRepository)(nil)

func validServiceProfile() serviceprofile.ServiceProfile {
	return serviceprofile.ServiceProfile{
		Name:   "Residential Internet",
		Status: serviceprofile.StatusActive,
	}
}

func TestServiceProfileServiceCreateSucceeds(t *testing.T) {
	repo := newFakeServiceProfileRepository()
	svc := service.NewServiceProfileService(repo)

	created, err := svc.Create(context.Background(), validServiceProfile())
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

func TestServiceProfileServiceCreateRejectsInvalidServiceProfileWithoutPersisting(t *testing.T) {
	repo := newFakeServiceProfileRepository()
	svc := service.NewServiceProfileService(repo)

	_, err := svc.Create(context.Background(), serviceprofile.ServiceProfile{}) // no Name, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestServiceProfileServiceUpdateSucceeds(t *testing.T) {
	existing := validServiceProfile()
	existing.ID = uuid.New()
	repo := newFakeServiceProfileRepository(existing)
	svc := service.NewServiceProfileService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = serviceprofile.StatusInactive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != serviceprofile.StatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, serviceprofile.StatusInactive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestServiceProfileServiceUpdateRejectsInvalidServiceProfileWithoutPersisting(t *testing.T) {
	existing := validServiceProfile()
	existing.ID = uuid.New()
	repo := newFakeServiceProfileRepository(existing)
	svc := service.NewServiceProfileService(repo)

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

func TestServiceProfileServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeServiceProfileRepository()
	svc := service.NewServiceProfileService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceProfileServiceListDelegatesToRepository(t *testing.T) {
	a := validServiceProfile()
	a.ID = uuid.New()
	b := validServiceProfile()
	b.ID = uuid.New()
	repo := newFakeServiceProfileRepository(a, b)
	svc := service.NewServiceProfileService(repo)

	profiles, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(profiles))
	}
}

func TestServiceProfileServiceDeleteSucceeds(t *testing.T) {
	existing := validServiceProfile()
	existing.ID = uuid.New()
	repo := newFakeServiceProfileRepository(existing)
	svc := service.NewServiceProfileService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceProfileServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeServiceProfileRepository()
	svc := service.NewServiceProfileService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
