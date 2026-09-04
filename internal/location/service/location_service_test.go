package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/location/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeLocationRepository is an in-memory location.LocationRepository. Like
// internal/customer/service/customer_service_test.go's
// fakeCustomerRepository, it exists so LocationService's business logic —
// validate, then delegate — is tested without a real database;
// internal/location/postgres/location_test.go already covers the
// repository itself against real PostgreSQL. It tracks whether
// Create/Update were actually invoked, which is what lets
// TestLocationServiceCreateRejectsInvalidLocationWithoutPersisting prove
// validation happens before any repository call.
type fakeLocationRepository struct {
	byID         map[uuid.UUID]location.Location
	createCalled bool
	updateCalled bool
}

func newFakeLocationRepository(locations ...location.Location) *fakeLocationRepository {
	f := &fakeLocationRepository{byID: make(map[uuid.UUID]location.Location)}
	for _, l := range locations {
		f.byID[l.ID] = l
	}
	return f
}

func (f *fakeLocationRepository) Get(_ context.Context, id uuid.UUID) (location.Location, error) {
	l, ok := f.byID[id]
	if !ok {
		return location.Location{}, apperror.NotFound("location not found")
	}
	return l, nil
}

func (f *fakeLocationRepository) List(_ context.Context) ([]location.Location, error) {
	locations := make([]location.Location, 0, len(f.byID))
	for _, l := range f.byID {
		locations = append(locations, l)
	}
	return locations, nil
}

func (f *fakeLocationRepository) Create(_ context.Context, l location.Location) (location.Location, error) {
	f.createCalled = true
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	f.byID[l.ID] = l
	return l, nil
}

func (f *fakeLocationRepository) Update(_ context.Context, l location.Location) (location.Location, error) {
	f.updateCalled = true
	if _, ok := f.byID[l.ID]; !ok {
		return location.Location{}, apperror.NotFound("location not found")
	}
	f.byID[l.ID] = l
	return l, nil
}

func (f *fakeLocationRepository) ListByCustomerID(_ context.Context, customerID uuid.UUID) ([]location.Location, error) {
	locations := []location.Location{}
	for _, l := range f.byID {
		if l.CustomerID == customerID {
			locations = append(locations, l)
		}
	}
	return locations, nil
}

func (f *fakeLocationRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("location not found")
	}
	delete(f.byID, id)
	return nil
}

var _ location.LocationRepository = (*fakeLocationRepository)(nil)

func validLocation() location.Location {
	return location.Location{
		CustomerID: uuid.New(),
		Name:       "Main Service Address",
		Type:       location.LocationTypeService,
		Status:     location.LocationStatusActive,
	}
}

func TestLocationServiceCreateSucceeds(t *testing.T) {
	repo := newFakeLocationRepository()
	svc := service.NewLocationService(repo)

	created, err := svc.Create(context.Background(), validLocation())
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

func TestLocationServiceCreateRejectsInvalidLocationWithoutPersisting(t *testing.T) {
	repo := newFakeLocationRepository()
	svc := service.NewLocationService(repo)

	_, err := svc.Create(context.Background(), location.Location{}) // no CustomerID, Name, Type, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestLocationServiceUpdateSucceeds(t *testing.T) {
	existing := validLocation()
	existing.ID = uuid.New()
	repo := newFakeLocationRepository(existing)
	svc := service.NewLocationService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = location.LocationStatusInactive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != location.LocationStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, location.LocationStatusInactive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestLocationServiceUpdateRejectsInvalidLocationWithoutPersisting(t *testing.T) {
	existing := validLocation()
	existing.ID = uuid.New()
	repo := newFakeLocationRepository(existing)
	svc := service.NewLocationService(repo)

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

func TestLocationServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeLocationRepository()
	svc := service.NewLocationService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestLocationServiceListDelegatesToRepository(t *testing.T) {
	a := validLocation()
	a.ID = uuid.New()
	b := validLocation()
	b.ID = uuid.New()
	repo := newFakeLocationRepository(a, b)
	svc := service.NewLocationService(repo)

	locations, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(locations))
	}
}

func TestLocationServiceDeleteSucceeds(t *testing.T) {
	existing := validLocation()
	existing.ID = uuid.New()
	repo := newFakeLocationRepository(existing)
	svc := service.NewLocationService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestLocationServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeLocationRepository()
	svc := service.NewLocationService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
