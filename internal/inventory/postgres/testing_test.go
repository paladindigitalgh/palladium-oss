//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// assertConflict checks that err is a platform apperror.KindConflict —
// the outcome every foreign key violation in this package translates to
// (see translateError in errors.go), whether from an insert/update
// referencing a nonexistent parent or a delete blocked by an existing
// child. assertNotFound, its KindNotFound counterpart, is already defined
// in site_test.go and reused as-is by every other test file in this
// package.
func assertConflict(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want a conflict error")
	}
	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindConflict, err)
	}
}

// newTestQuerier opens a transaction against the real test database and
// returns it as a database.Querier, rolled back automatically on cleanup.
// It is the shared foundation every entity's newXTestRepository helper is
// built on (see e.g. newBuildingTestRepository in building_test.go), and is
// also used directly by tests that need two repositories sharing one
// transaction — e.g. a parent-delete-blocked-by-child test needs both the
// parent's and the child's repository to see the same uncommitted rows.
//
// This duplicates a few lines of testConfig/connect/BeginTx setup that
// already exists inline in site_test.go's newTestRepository. That is
// deliberate, not an oversight: this milestone's instructions are to
// extend the Site pattern to new entities without refactoring Site, so
// site_test.go is left untouched rather than pulled onto this shared
// helper.
func newTestQuerier(t *testing.T) (database.Querier, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := database.Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping() = %v; is Postgres running and migrated? try `make db-up && make migrate-up`", err)
	}

	tx, err := pool.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() = %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	return tx, ctx
}

// The fixture helpers below build a valid parent chain so a child entity's
// tests can satisfy its required (or, for exercising the non-nil case, its
// nullable) foreign key without every test re-deriving the same setup.
// Each creates real rows through the already-tested repository for its
// level, not by hand-crafting SQL, so a fixture failure surfaces as a
// clear failure of that repository's own Create rather than a confusing
// failure somewhere else.

func createTestSite(t *testing.T, ctx context.Context, q database.Querier) inventory.Site {
	t.Helper()

	repo := postgres.NewSiteRepository(q, clock.New(), id.New())
	site, err := repo.Create(ctx, inventory.Site{
		Metadata: inventory.Metadata{Name: "Fixture Site " + uuid.NewString()},
	})
	if err != nil {
		t.Fatalf("fixture: create site: %v", err)
	}
	return site
}

func createTestBuilding(t *testing.T, ctx context.Context, q database.Querier) inventory.Building {
	t.Helper()

	site := createTestSite(t, ctx, q)
	repo := postgres.NewBuildingRepository(q, clock.New(), id.New())
	building, err := repo.Create(ctx, inventory.Building{
		Metadata: inventory.Metadata{Name: "Fixture Building " + uuid.NewString()},
		SiteID:   site.ID,
	})
	if err != nil {
		t.Fatalf("fixture: create building: %v", err)
	}
	return building
}

func createTestRoom(t *testing.T, ctx context.Context, q database.Querier) inventory.Room {
	t.Helper()

	building := createTestBuilding(t, ctx, q)
	repo := postgres.NewRoomRepository(q, clock.New(), id.New())
	room, err := repo.Create(ctx, inventory.Room{
		Metadata:   inventory.Metadata{Name: "Fixture Room " + uuid.NewString()},
		BuildingID: building.ID,
	})
	if err != nil {
		t.Fatalf("fixture: create room: %v", err)
	}
	return room
}

func createTestRack(t *testing.T, ctx context.Context, q database.Querier) inventory.Rack {
	t.Helper()

	room := createTestRoom(t, ctx, q)
	roomID := room.ID
	repo := postgres.NewRackRepository(q, clock.New(), id.New())
	rack, err := repo.Create(ctx, inventory.Rack{
		Metadata: inventory.Metadata{Name: "Fixture Rack " + uuid.NewString()},
		RoomID:   &roomID,
	})
	if err != nil {
		t.Fatalf("fixture: create rack: %v", err)
	}
	return rack
}
