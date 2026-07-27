package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeSiteRepository is an in-memory inventory.SiteRepository. Like
// internal/auth/service_test.go's fakeUserRepository, it exists so
// SiteService's business logic — validate, then delegate — is tested
// without a real database; internal/inventory/postgres/site_test.go
// already covers the repository itself against real PostgreSQL. It
// tracks whether Create/Update were actually invoked, which is what lets
// TestSiteServiceCreateRejectsInvalidSiteWithoutPersisting prove
// validation happens before any repository call, not just that the
// overall result is an error.
type fakeSiteRepository struct {
	byID         map[uuid.UUID]inventory.Site
	createCalled bool
	updateCalled bool
}

func newFakeSiteRepository(sites ...inventory.Site) *fakeSiteRepository {
	f := &fakeSiteRepository{byID: make(map[uuid.UUID]inventory.Site)}
	for _, s := range sites {
		f.byID[s.ID] = s
	}
	return f
}

func (f *fakeSiteRepository) Get(_ context.Context, id uuid.UUID) (inventory.Site, error) {
	s, ok := f.byID[id]
	if !ok {
		return inventory.Site{}, apperror.NotFound("site not found")
	}
	return s, nil
}

func (f *fakeSiteRepository) List(_ context.Context) ([]inventory.Site, error) {
	sites := make([]inventory.Site, 0, len(f.byID))
	for _, s := range f.byID {
		sites = append(sites, s)
	}
	return sites, nil
}

func (f *fakeSiteRepository) Create(_ context.Context, site inventory.Site) (inventory.Site, error) {
	f.createCalled = true
	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	f.byID[site.ID] = site
	return site, nil
}

func (f *fakeSiteRepository) Update(_ context.Context, site inventory.Site) (inventory.Site, error) {
	f.updateCalled = true
	if _, ok := f.byID[site.ID]; !ok {
		return inventory.Site{}, apperror.NotFound("site not found")
	}
	f.byID[site.ID] = site
	return site, nil
}

func (f *fakeSiteRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("site not found")
	}
	delete(f.byID, id)
	return nil
}

var _ inventory.SiteRepository = (*fakeSiteRepository)(nil)

func TestSiteServiceCreateSucceeds(t *testing.T) {
	repo := newFakeSiteRepository()
	svc := service.NewSiteService(repo)

	site := inventory.Site{Metadata: inventory.Metadata{Name: "Main Office"}}

	created, err := svc.Create(context.Background(), site)
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

func TestSiteServiceCreateRejectsInvalidSiteWithoutPersisting(t *testing.T) {
	repo := newFakeSiteRepository()
	svc := service.NewSiteService(repo)

	_, err := svc.Create(context.Background(), inventory.Site{}) // no Name

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestSiteServiceUpdateSucceeds(t *testing.T) {
	existing := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}}
	repo := newFakeSiteRepository(existing)
	svc := service.NewSiteService(repo)

	updated, err := svc.Update(context.Background(), inventory.Site{
		Metadata: inventory.Metadata{ID: existing.ID, Name: "New Name"},
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestSiteServiceUpdateRejectsInvalidSiteWithoutPersisting(t *testing.T) {
	existing := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}}
	repo := newFakeSiteRepository(existing)
	svc := service.NewSiteService(repo)

	_, err := svc.Update(context.Background(), inventory.Site{
		Metadata: inventory.Metadata{ID: existing.ID, Name: ""}, // invalid
	})

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestSiteServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeSiteRepository()
	svc := service.NewSiteService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestSiteServiceListDelegatesToRepository(t *testing.T) {
	a := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}}
	b := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}}
	repo := newFakeSiteRepository(a, b)
	svc := service.NewSiteService(repo)

	sites, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(sites) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(sites))
	}
}

func TestSiteServiceDeleteSucceeds(t *testing.T) {
	existing := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}}
	repo := newFakeSiteRepository(existing)
	svc := service.NewSiteService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestSiteServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeSiteRepository()
	svc := service.NewSiteService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
