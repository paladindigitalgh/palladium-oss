package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeBuildingRepository is an in-memory inventory.BuildingRepository. See
// fakeSiteRepository's doc comment in site_service_test.go for why this
// exists (and why it tracks createCalled/updateCalled).
type fakeBuildingRepository struct {
	byID         map[uuid.UUID]inventory.Building
	createCalled bool
	updateCalled bool
}

func newFakeBuildingRepository(buildings ...inventory.Building) *fakeBuildingRepository {
	f := &fakeBuildingRepository{byID: make(map[uuid.UUID]inventory.Building)}
	for _, b := range buildings {
		f.byID[b.ID] = b
	}
	return f
}

func (f *fakeBuildingRepository) Get(_ context.Context, id uuid.UUID) (inventory.Building, error) {
	b, ok := f.byID[id]
	if !ok {
		return inventory.Building{}, apperror.NotFound("building not found")
	}
	return b, nil
}

func (f *fakeBuildingRepository) List(_ context.Context) ([]inventory.Building, error) {
	buildings := make([]inventory.Building, 0, len(f.byID))
	for _, b := range f.byID {
		buildings = append(buildings, b)
	}
	return buildings, nil
}

func (f *fakeBuildingRepository) Create(_ context.Context, building inventory.Building) (inventory.Building, error) {
	f.createCalled = true
	if building.ID == uuid.Nil {
		building.ID = uuid.New()
	}
	f.byID[building.ID] = building
	return building, nil
}

func (f *fakeBuildingRepository) Update(_ context.Context, building inventory.Building) (inventory.Building, error) {
	f.updateCalled = true
	if _, ok := f.byID[building.ID]; !ok {
		return inventory.Building{}, apperror.NotFound("building not found")
	}
	f.byID[building.ID] = building
	return building, nil
}

func (f *fakeBuildingRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("building not found")
	}
	delete(f.byID, id)
	return nil
}

var _ inventory.BuildingRepository = (*fakeBuildingRepository)(nil)

func TestBuildingServiceCreateSucceeds(t *testing.T) {
	repo := newFakeBuildingRepository()
	svc := service.NewBuildingService(repo)

	building := inventory.Building{Metadata: inventory.Metadata{Name: "Main Office"}, SiteID: uuid.New()}

	created, err := svc.Create(context.Background(), building)
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

func TestBuildingServiceCreateRejectsInvalidBuildingWithoutPersisting(t *testing.T) {
	repo := newFakeBuildingRepository()
	svc := service.NewBuildingService(repo)

	_, err := svc.Create(context.Background(), inventory.Building{}) // no Name, no SiteID

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestBuildingServiceUpdateSucceeds(t *testing.T) {
	siteID := uuid.New()
	existing := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}, SiteID: siteID}
	repo := newFakeBuildingRepository(existing)
	svc := service.NewBuildingService(repo)

	updated, err := svc.Update(context.Background(), inventory.Building{
		Metadata: inventory.Metadata{ID: existing.ID, Name: "New Name"},
		SiteID:   siteID,
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

func TestBuildingServiceUpdateRejectsInvalidBuildingWithoutPersisting(t *testing.T) {
	existing := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}, SiteID: uuid.New()}
	repo := newFakeBuildingRepository(existing)
	svc := service.NewBuildingService(repo)

	_, err := svc.Update(context.Background(), inventory.Building{
		Metadata: inventory.Metadata{ID: existing.ID, Name: ""}, // invalid
		SiteID:   existing.SiteID,
	})

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestBuildingServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeBuildingRepository()
	svc := service.NewBuildingService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestBuildingServiceListDelegatesToRepository(t *testing.T) {
	a := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}, SiteID: uuid.New()}
	b := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}, SiteID: uuid.New()}
	repo := newFakeBuildingRepository(a, b)
	svc := service.NewBuildingService(repo)

	buildings, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(buildings) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(buildings))
	}
}

func TestBuildingServiceDeleteSucceeds(t *testing.T) {
	existing := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}, SiteID: uuid.New()}
	repo := newFakeBuildingRepository(existing)
	svc := service.NewBuildingService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestBuildingServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeBuildingRepository()
	svc := service.NewBuildingService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
