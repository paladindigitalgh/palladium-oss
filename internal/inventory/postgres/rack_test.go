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

func TestRackRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	room := createTestRoom(t, ctx, q)
	roomID := room.ID
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Rack{
		Metadata: inventory.Metadata{Name: "Rack 1", Description: "42U"},
		RoomID:   &roomID,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.RoomID == nil || *created.RoomID != room.ID {
		t.Errorf("RoomID = %v, want %v", created.RoomID, room.ID)
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

// TestRackRepositoryCreateWithoutRoom exercises the nullable relationship:
// a Rack can exist in inventory before it is installed in a Room (see
// inventory.Rack in internal/inventory/model.go).
func TestRackRepositoryCreateWithoutRoom(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "Spare Rack"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.RoomID != nil {
		t.Errorf("RoomID = %v, want nil", created.RoomID)
	}
}

func TestRackRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	created, err := repo.Create(ctx, inventory.Rack{
		Metadata: inventory.Metadata{ID: bogusID, Name: "Edge Rack", CreatedAt: bogusTime, UpdatedAt: bogusTime},
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

func TestRackRepositoryCreateFailsWhenRoomDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	bogusRoomID := uuid.New() // does not exist
	_, err := repo.Create(ctx, inventory.Rack{
		Metadata: inventory.Metadata{Name: "Orphan Rack"},
		RoomID:   &bogusRoomID,
	})

	assertConflict(t, err)
}

func TestRackRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "Rack A"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if got.RoomID != nil {
		t.Errorf("RoomID = %v, want nil", got.RoomID)
	}
}

func TestRackRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestRackRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "Alpha Rack"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "Beta Rack"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	racks, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]inventory.Rack, len(racks))
	for _, r := range racks {
		found[r.ID] = r
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created rack")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created rack")
	}

	if len(racks) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(racks), racks)
	}
	if racks[0].Name != "Alpha Rack" || racks[1].Name != "Beta Rack" {
		t.Errorf("List() order = [%q, %q], want [Alpha Rack, Beta Rack]", racks[0].Name, racks[1].Name)
	}
}

// TestRackRepositoryUpdateInstallsAndUninstalls exercises RoomID moving in
// both directions: nil -> set (a rack being installed) and set -> nil (a
// rack being pulled from service), proving Update treats the nullable
// foreign key as fully mutable like every other field.
func TestRackRepositoryUpdateInstallsAndUninstalls(t *testing.T) {
	q, ctx := newTestQuerier(t)
	room := createTestRoom(t, ctx, q)
	roomID := room.ID
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "Mobile Rack"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.RoomID != nil {
		t.Fatalf("precondition failed: RoomID = %v, want nil", created.RoomID)
	}

	installed, err := repo.Update(ctx, inventory.Rack{
		Metadata: inventory.Metadata{ID: created.ID, Name: created.Name},
		RoomID:   &roomID,
	})
	if err != nil {
		t.Fatalf("Update() (install) = %v", err)
	}
	if installed.RoomID == nil || *installed.RoomID != room.ID {
		t.Errorf("RoomID = %v, want %v", installed.RoomID, room.ID)
	}
	if !installed.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, installed.CreatedAt)
	}

	uninstalled, err := repo.Update(ctx, inventory.Rack{
		Metadata: inventory.Metadata{ID: created.ID, Name: created.Name},
		RoomID:   nil,
	})
	if err != nil {
		t.Fatalf("Update() (uninstall) = %v", err)
	}
	if uninstalled.RoomID != nil {
		t.Errorf("RoomID = %v, want nil", uninstalled.RoomID)
	}
}

func TestRackRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	_, err := repo.Update(ctx, inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Ghost"}})

	assertNotFound(t, err)
}

func TestRackRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "Temporary"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestRackRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewRackRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestRackRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	fixedID := uuid.New()
	repo := postgres.NewRackRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "First"}}); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, inventory.Rack{Metadata: inventory.Metadata{Name: "Second"}})
	assertConflict(t, err)
}

// TestRoomRepositoryDeleteBlockedByExistingRack mirrors
// TestBuildingRepositoryDeleteBlockedByExistingRoom in room_test.go, one
// level down the hierarchy. It only demonstrates the block when the Rack
// is actually installed in the Room (non-nil RoomID) — an unracked Rack
// has nothing to block on, which is exactly the point of the nullable
// relationship.
func TestRoomRepositoryDeleteBlockedByExistingRack(t *testing.T) {
	q, ctx := newTestQuerier(t)
	rack := createTestRack(t, ctx, q)
	roomRepo := postgres.NewRoomRepository(q, clock.New(), id.New())

	err := roomRepo.Delete(ctx, *rack.RoomID)

	assertConflict(t, err)
}
