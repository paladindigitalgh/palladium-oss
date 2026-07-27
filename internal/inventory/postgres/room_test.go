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

func TestRoomRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Room{
		Metadata:   inventory.Metadata{Name: "Server Room", Description: "Ground floor"},
		BuildingID: building.ID,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.BuildingID != building.ID {
		t.Errorf("BuildingID = %v, want %v", created.BuildingID, building.ID)
	}
	if created.Name != "Server Room" {
		t.Errorf("Name = %q, want %q", created.Name, "Server Room")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestRoomRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	created, err := repo.Create(ctx, inventory.Room{
		Metadata:   inventory.Metadata{ID: bogusID, Name: "Edge Room", CreatedAt: bogusTime, UpdatedAt: bogusTime},
		BuildingID: building.ID,
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

func TestRoomRepositoryCreateFailsWhenBuildingDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, inventory.Room{
		Metadata:   inventory.Metadata{Name: "Orphan Room"},
		BuildingID: uuid.New(), // does not exist
	})

	assertConflict(t, err)
}

func TestRoomRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Room{Metadata: inventory.Metadata{Name: "POP 1"}, BuildingID: building.ID})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.BuildingID != created.BuildingID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
}

func TestRoomRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestRoomRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, inventory.Room{Metadata: inventory.Metadata{Name: "Alpha Room"}, BuildingID: building.ID})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, inventory.Room{Metadata: inventory.Metadata{Name: "Beta Room"}, BuildingID: building.ID})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	rooms, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]inventory.Room, len(rooms))
	for _, r := range rooms {
		found[r.ID] = r
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created room")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created room")
	}

	if len(rooms) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(rooms), rooms)
	}
	if rooms[0].Name != "Alpha Room" || rooms[1].Name != "Beta Room" {
		t.Errorf("List() order = [%q, %q], want [Alpha Room, Beta Room]", rooms[0].Name, rooms[1].Name)
	}
}

func TestRoomRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	otherBuilding := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Room{
		Metadata:   inventory.Metadata{Name: "Old Name", Description: "Old Description"},
		BuildingID: building.ID,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, inventory.Room{
		Metadata:   inventory.Metadata{ID: created.ID, Name: "New Name", Description: "New Description"},
		BuildingID: otherBuilding.ID,
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.BuildingID != otherBuilding.ID {
		t.Errorf("BuildingID = %v, want %v (BuildingID must be mutable via Update)", updated.BuildingID, otherBuilding.ID)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestRoomRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	_, err := repo.Update(ctx, inventory.Room{
		Metadata:   inventory.Metadata{ID: uuid.New(), Name: "Ghost"},
		BuildingID: building.ID,
	})

	assertNotFound(t, err)
}

func TestRoomRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Room{Metadata: inventory.Metadata{Name: "Temporary"}, BuildingID: building.ID})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestRoomRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestRoomRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	building := createTestBuilding(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewRoomRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, inventory.Room{Metadata: inventory.Metadata{Name: "First"}, BuildingID: building.ID}); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, inventory.Room{Metadata: inventory.Metadata{Name: "Second"}, BuildingID: building.ID})
	assertConflict(t, err)
}

// TestBuildingRepositoryDeleteBlockedByExistingRoom mirrors
// TestSiteRepositoryDeleteBlockedByExistingBuilding in building_test.go,
// one level down the hierarchy.
func TestBuildingRepositoryDeleteBlockedByExistingRoom(t *testing.T) {
	q, ctx := newTestQuerier(t)
	room := createTestRoom(t, ctx, q)
	buildingRepo := postgres.NewBuildingRepository(q, clock.New(), id.New())

	err := buildingRepo.Delete(ctx, room.BuildingID)

	assertConflict(t, err)
}
