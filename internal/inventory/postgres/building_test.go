//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// Unlike Site, every Building test needs a fixture Site to satisfy the
// required SiteID foreign key, and the fixture must share the same
// transaction as the repository under test (see createTestSite in
// testing_test.go). So tests here call newTestQuerier directly and build
// both the fixture and the repository under test from the same Querier,
// rather than hiding it behind a Site-style newTestRepository wrapper.

func TestBuildingRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Building{
		Metadata: inventory.Metadata{Name: "Main Office", Description: "HQ"},
		SiteID:   site.ID,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.SiteID != site.ID {
		t.Errorf("SiteID = %v, want %v", created.SiteID, site.ID)
	}
	if created.Name != "Main Office" {
		t.Errorf("Name = %q, want %q", created.Name, "Main Office")
	}
	if created.Description != "HQ" {
		t.Errorf("Description = %q, want %q", created.Description, "HQ")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestBuildingRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	created, err := repo.Create(ctx, inventory.Building{
		Metadata: inventory.Metadata{
			ID:        bogusID,
			Name:      "Edge Building",
			CreatedAt: bogusTime,
			UpdatedAt: bogusTime,
		},
		SiteID: site.ID,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == bogusID {
		t.Error("Create() used the caller-supplied ID instead of generating one")
	}
	if created.CreatedAt.Equal(bogusTime) {
		t.Error("Create() used the caller-supplied CreatedAt instead of stamping the current time")
	}
}

func TestBuildingRepositoryCreateFailsWhenSiteDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, inventory.Building{
		Metadata: inventory.Metadata{Name: "Orphan Building"},
		SiteID:   uuid.New(), // does not exist
	})

	assertConflict(t, err)
}

func TestBuildingRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Building{
		Metadata: inventory.Metadata{Name: "POP 1"},
		SiteID:   site.ID,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.SiteID != created.SiteID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
}

func TestBuildingRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestBuildingRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, inventory.Building{Metadata: inventory.Metadata{Name: "Alpha Building"}, SiteID: site.ID})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, inventory.Building{Metadata: inventory.Metadata{Name: "Beta Building"}, SiteID: site.ID})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	buildings, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]inventory.Building, len(buildings))
	for _, b := range buildings {
		found[b.ID] = b
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created building")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created building")
	}

	if len(buildings) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(buildings), buildings)
	}
	if buildings[0].Name != "Alpha Building" || buildings[1].Name != "Beta Building" {
		t.Errorf("List() order = [%q, %q], want [Alpha Building, Beta Building]", buildings[0].Name, buildings[1].Name)
	}
}

func TestBuildingRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	otherSite := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Building{
		Metadata: inventory.Metadata{Name: "Old Name", Description: "Old Description"},
		SiteID:   site.ID,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, inventory.Building{
		Metadata: inventory.Metadata{ID: created.ID, Name: "New Name", Description: "New Description"},
		SiteID:   otherSite.ID,
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Description != "New Description" {
		t.Errorf("Description = %q, want %q", updated.Description, "New Description")
	}
	if updated.SiteID != otherSite.ID {
		t.Errorf("SiteID = %v, want %v (SiteID must be mutable via Update)", updated.SiteID, otherSite.ID)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestBuildingRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	_, err := repo.Update(ctx, inventory.Building{
		Metadata: inventory.Metadata{ID: uuid.New(), Name: "Ghost"},
		SiteID:   site.ID,
	})

	assertNotFound(t, err)
}

func TestBuildingRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Building{Metadata: inventory.Metadata{Name: "Temporary"}, SiteID: site.ID})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestBuildingRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestBuildingRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	site := createTestSite(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewBuildingRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, inventory.Building{Metadata: inventory.Metadata{Name: "First"}, SiteID: site.ID}); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, inventory.Building{Metadata: inventory.Metadata{Name: "Second"}, SiteID: site.ID})
	assertConflict(t, err)
}

// TestSiteRepositoryDeleteBlockedByExistingBuilding lives here, not in
// site_test.go, so that file stays untouched (this milestone's
// instructions are to extend the Site pattern without refactoring Site).
// It exercises SiteRepository.Delete against the new foreign key that this
// milestone's building migration adds.
func TestSiteRepositoryDeleteBlockedByExistingBuilding(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	siteRepo := postgres.NewSiteRepository(q, clock.New(), id.New())

	err := siteRepo.Delete(ctx, building.SiteID)

	assertConflict(t, err)
}
