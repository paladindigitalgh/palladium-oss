package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	"github.com/paladindigitalgh/palladium-oss/internal/catalog/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeCatalogRepository is an in-memory catalog.CatalogRepository. Like
// internal/location/service/location_service_test.go's
// fakeLocationRepository, it exists so CatalogService's business logic —
// validate, then delegate — is tested without a real database;
// internal/catalog/postgres/catalog_test.go already covers the repository
// itself against real PostgreSQL. It tracks whether Create/Update were
// actually invoked, which is what lets
// TestCatalogServiceCreateRejectsInvalidCatalogWithoutPersisting prove
// validation happens before any repository call.
type fakeCatalogRepository struct {
	byID         map[uuid.UUID]catalog.ProductCatalog
	createCalled bool
	updateCalled bool
}

func newFakeCatalogRepository(catalogs ...catalog.ProductCatalog) *fakeCatalogRepository {
	f := &fakeCatalogRepository{byID: make(map[uuid.UUID]catalog.ProductCatalog)}
	for _, c := range catalogs {
		f.byID[c.ID] = c
	}
	return f
}

func (f *fakeCatalogRepository) Get(_ context.Context, id uuid.UUID) (catalog.ProductCatalog, error) {
	c, ok := f.byID[id]
	if !ok {
		return catalog.ProductCatalog{}, apperror.NotFound("catalog not found")
	}
	return c, nil
}

func (f *fakeCatalogRepository) List(_ context.Context) ([]catalog.ProductCatalog, error) {
	catalogs := make([]catalog.ProductCatalog, 0, len(f.byID))
	for _, c := range f.byID {
		catalogs = append(catalogs, c)
	}
	return catalogs, nil
}

func (f *fakeCatalogRepository) Create(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	f.createCalled = true
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeCatalogRepository) Update(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	f.updateCalled = true
	if _, ok := f.byID[c.ID]; !ok {
		return catalog.ProductCatalog{}, apperror.NotFound("catalog not found")
	}
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeCatalogRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("catalog not found")
	}
	delete(f.byID, id)
	return nil
}

var _ catalog.CatalogRepository = (*fakeCatalogRepository)(nil)

func validCatalog() catalog.ProductCatalog {
	return catalog.ProductCatalog{
		Name:   "Residential",
		Status: catalog.CatalogStatusActive,
	}
}

func TestCatalogServiceCreateSucceeds(t *testing.T) {
	repo := newFakeCatalogRepository()
	svc := service.NewCatalogService(repo)

	created, err := svc.Create(context.Background(), validCatalog())
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

func TestCatalogServiceCreateRejectsInvalidCatalogWithoutPersisting(t *testing.T) {
	repo := newFakeCatalogRepository()
	svc := service.NewCatalogService(repo)

	_, err := svc.Create(context.Background(), catalog.ProductCatalog{}) // no Name, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestCatalogServiceUpdateSucceeds(t *testing.T) {
	existing := validCatalog()
	existing.ID = uuid.New()
	repo := newFakeCatalogRepository(existing)
	svc := service.NewCatalogService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = catalog.CatalogStatusInactive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != catalog.CatalogStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, catalog.CatalogStatusInactive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestCatalogServiceUpdateRejectsInvalidCatalogWithoutPersisting(t *testing.T) {
	existing := validCatalog()
	existing.ID = uuid.New()
	repo := newFakeCatalogRepository(existing)
	svc := service.NewCatalogService(repo)

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

func TestCatalogServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeCatalogRepository()
	svc := service.NewCatalogService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestCatalogServiceListDelegatesToRepository(t *testing.T) {
	a := validCatalog()
	a.ID = uuid.New()
	b := validCatalog()
	b.ID = uuid.New()
	repo := newFakeCatalogRepository(a, b)
	svc := service.NewCatalogService(repo)

	catalogs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(catalogs) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(catalogs))
	}
}

func TestCatalogServiceDeleteSucceeds(t *testing.T) {
	existing := validCatalog()
	existing.ID = uuid.New()
	repo := newFakeCatalogRepository(existing)
	svc := service.NewCatalogService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestCatalogServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeCatalogRepository()
	svc := service.NewCatalogService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
