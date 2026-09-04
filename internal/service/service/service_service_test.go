package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/service/service"
)

// fakeServiceRepository is an in-memory domainservice.ServiceRepository.
// Like internal/product/service/product_service_test.go's
// fakeProductRepository, it exists so ServiceService's business logic —
// validate, then delegate — is tested without a real database;
// internal/service/postgres/service_test.go already covers the
// repository itself against real PostgreSQL. It tracks whether
// Create/Update were actually invoked, which is what lets
// TestServiceServiceCreateRejectsInvalidServiceWithoutPersisting prove
// validation happens before any repository call.
type fakeServiceRepository struct {
	byID         map[uuid.UUID]domainservice.Service
	createCalled bool
	updateCalled bool
}

func newFakeServiceRepository(services ...domainservice.Service) *fakeServiceRepository {
	f := &fakeServiceRepository{byID: make(map[uuid.UUID]domainservice.Service)}
	for _, s := range services {
		f.byID[s.ID] = s
	}
	return f
}

func (f *fakeServiceRepository) Get(_ context.Context, id uuid.UUID) (domainservice.Service, error) {
	s, ok := f.byID[id]
	if !ok {
		return domainservice.Service{}, apperror.NotFound("service not found")
	}
	return s, nil
}

func (f *fakeServiceRepository) List(_ context.Context) ([]domainservice.Service, error) {
	services := make([]domainservice.Service, 0, len(f.byID))
	for _, s := range f.byID {
		services = append(services, s)
	}
	return services, nil
}

func (f *fakeServiceRepository) Create(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	f.createCalled = true
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	f.byID[s.ID] = s
	return s, nil
}

func (f *fakeServiceRepository) Update(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	f.updateCalled = true
	if _, ok := f.byID[s.ID]; !ok {
		return domainservice.Service{}, apperror.NotFound("service not found")
	}
	f.byID[s.ID] = s
	return s, nil
}

func (f *fakeServiceRepository) ListByLocationID(_ context.Context, locationID uuid.UUID) ([]domainservice.Service, error) {
	services := []domainservice.Service{}
	for _, s := range f.byID {
		if s.LocationID == locationID {
			services = append(services, s)
		}
	}
	return services, nil
}

func (f *fakeServiceRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("service not found")
	}
	delete(f.byID, id)
	return nil
}

var _ domainservice.ServiceRepository = (*fakeServiceRepository)(nil)

func validService() domainservice.Service {
	return domainservice.Service{
		LocationID:       uuid.New(),
		ProductID:        uuid.New(),
		ServiceProfileID: uuid.New(),
		Status:           domainservice.ServiceStatusPending,
	}
}

func TestServiceServiceCreateSucceeds(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo)

	created, err := svc.Create(context.Background(), validService())
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

func TestServiceServiceCreateRejectsInvalidServiceWithoutPersisting(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo)

	_, err := svc.Create(context.Background(), domainservice.Service{}) // no LocationID, ProductID, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestServiceServiceUpdateSucceeds(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	repo := newFakeServiceRepository(existing)
	svc := service.NewServiceService(repo)

	toUpdate := existing
	toUpdate.Status = domainservice.ServiceStatusActive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Status != domainservice.ServiceStatusActive {
		t.Errorf("Status = %q, want %q", updated.Status, domainservice.ServiceStatusActive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestServiceServiceUpdateRejectsInvalidServiceWithoutPersisting(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	repo := newFakeServiceRepository(existing)
	svc := service.NewServiceService(repo)

	invalid := existing
	invalid.Status = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestServiceServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceServiceListDelegatesToRepository(t *testing.T) {
	a := validService()
	a.ID = uuid.New()
	b := validService()
	b.ID = uuid.New()
	repo := newFakeServiceRepository(a, b)
	svc := service.NewServiceService(repo)

	services, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(services))
	}
}

func TestServiceServiceDeleteSucceeds(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	repo := newFakeServiceRepository(existing)
	svc := service.NewServiceService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
