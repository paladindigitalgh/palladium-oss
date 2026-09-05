package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
	"github.com/paladindigitalgh/palladium-oss/internal/provider/service"
)

// fakeProviderRepository is an in-memory provider.ProviderRepository.
// Like internal/serviceprofile/service/service_profile_service_test.go's
// fakeServiceProfileRepository, it exists so ProviderService's business
// logic — validate, then delegate — is tested without a real database;
// internal/provider/postgres/provider_test.go already covers the
// repository itself against real PostgreSQL.
type fakeProviderRepository struct {
	byID         map[uuid.UUID]provider.Provider
	createCalled bool
	updateCalled bool
}

func newFakeProviderRepository(providers ...provider.Provider) *fakeProviderRepository {
	f := &fakeProviderRepository{byID: make(map[uuid.UUID]provider.Provider)}
	for _, p := range providers {
		f.byID[p.ID] = p
	}
	return f
}

func (f *fakeProviderRepository) Get(_ context.Context, id uuid.UUID) (provider.Provider, error) {
	p, ok := f.byID[id]
	if !ok {
		return provider.Provider{}, apperror.NotFound("provider not found")
	}
	return p, nil
}

func (f *fakeProviderRepository) List(_ context.Context) ([]provider.Provider, error) {
	providers := make([]provider.Provider, 0, len(f.byID))
	for _, p := range f.byID {
		providers = append(providers, p)
	}
	return providers, nil
}

func (f *fakeProviderRepository) Create(_ context.Context, p provider.Provider) (provider.Provider, error) {
	f.createCalled = true
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeProviderRepository) Update(_ context.Context, p provider.Provider) (provider.Provider, error) {
	f.updateCalled = true
	if _, ok := f.byID[p.ID]; !ok {
		return provider.Provider{}, apperror.NotFound("provider not found")
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeProviderRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("provider not found")
	}
	delete(f.byID, id)
	return nil
}

var _ provider.ProviderRepository = (*fakeProviderRepository)(nil)

func validProvider() provider.Provider {
	return provider.Provider{
		Name:   "Acme Fiber",
		Status: provider.StatusActive,
	}
}

func TestProviderServiceCreateSucceeds(t *testing.T) {
	repo := newFakeProviderRepository()
	svc := service.NewProviderService(repo)

	created, err := svc.Create(context.Background(), validProvider())
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

func TestProviderServiceCreateRejectsInvalidProviderWithoutPersisting(t *testing.T) {
	repo := newFakeProviderRepository()
	svc := service.NewProviderService(repo)

	_, err := svc.Create(context.Background(), provider.Provider{}) // no Name, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestProviderServiceUpdateSucceeds(t *testing.T) {
	existing := validProvider()
	existing.ID = uuid.New()
	repo := newFakeProviderRepository(existing)
	svc := service.NewProviderService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = provider.StatusInactive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != provider.StatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, provider.StatusInactive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestProviderServiceUpdateRejectsInvalidProviderWithoutPersisting(t *testing.T) {
	existing := validProvider()
	existing.ID = uuid.New()
	repo := newFakeProviderRepository(existing)
	svc := service.NewProviderService(repo)

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

func TestProviderServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeProviderRepository()
	svc := service.NewProviderService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProviderServiceListDelegatesToRepository(t *testing.T) {
	a := validProvider()
	a.ID = uuid.New()
	b := validProvider()
	b.ID = uuid.New()
	repo := newFakeProviderRepository(a, b)
	svc := service.NewProviderService(repo)

	providers, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(providers))
	}
}

func TestProviderServiceDeleteSucceeds(t *testing.T) {
	existing := validProvider()
	existing.ID = uuid.New()
	repo := newFakeProviderRepository(existing)
	svc := service.NewProviderService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProviderServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeProviderRepository()
	svc := service.NewProviderService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
