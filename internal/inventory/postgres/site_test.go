//go:build integration

package postgres_test

import (
	"context"
	"os"
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

// newTestRepository opens a transaction against the real test database and
// builds a SiteRepository on top of it (not the pool directly). The
// transaction is always rolled back on cleanup, so every test starts from a
// clean slate without needing manual DELETE/TRUNCATE bookkeeping, and tests
// never observe each other's writes.
//
// This is also a live demonstration of why SiteRepository depends on
// database.Querier instead of *database.Pool: the exact same repository
// code runs here on a pgx.Tx and in production on the pool.
//
// ids lets Create-conflict tests inject a generator that returns a fixed
// UUID; every other test should pass id.New().
func newTestRepository(t *testing.T, ids id.Generator) (*postgres.SiteRepository, context.Context) {
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

	return postgres.NewSiteRepository(tx, clock.New(), ids), ctx
}

func TestSiteRepositoryCreate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, inventory.Site{
		Metadata: inventory.Metadata{Name: "Main Office", Description: "HQ"},
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
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

func TestSiteRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	created, err := repo.Create(ctx, inventory.Site{
		Metadata: inventory.Metadata{
			ID:        bogusID,
			Name:      "Edge Site",
			CreatedAt: bogusTime,
			UpdatedAt: bogusTime,
		},
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

func TestSiteRepositoryGet(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, inventory.Site{Metadata: inventory.Metadata{Name: "POP 1"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	// Compared field by field rather than with == or reflect.DeepEqual:
	// time.Time is documented as unsafe to compare that way (it embeds an
	// optional monotonic reading), so timestamps use Equal explicitly.
	if got.ID != created.ID || got.Name != created.Name || got.Description != created.Description {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestSiteRepositoryGetNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestSiteRepositoryList(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	first, err := repo.Create(ctx, inventory.Site{Metadata: inventory.Metadata{Name: "Alpha Hub"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, inventory.Site{Metadata: inventory.Metadata{Name: "Beta Hub"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	sites, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]inventory.Site, len(sites))
	for _, s := range sites {
		found[s.ID] = s
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created site")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created site")
	}

	// Both were created within this same rolled-back transaction, so the
	// list is exactly these two, letting us also check the ORDER BY name.
	if len(sites) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(sites), sites)
	}
	if sites[0].Name != "Alpha Hub" || sites[1].Name != "Beta Hub" {
		t.Errorf("List() order = [%q, %q], want [Alpha Hub, Beta Hub]", sites[0].Name, sites[1].Name)
	}
}

func TestSiteRepositoryUpdate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, inventory.Site{
		Metadata: inventory.Metadata{Name: "Old Name", Description: "Old Description"},
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, inventory.Site{
		Metadata: inventory.Metadata{
			ID:          created.ID,
			Name:        "New Name",
			Description: "New Description",
		},
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
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestSiteRepositoryUpdateNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Update(ctx, inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Ghost"}})

	assertNotFound(t, err)
}

func TestSiteRepositoryDelete(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, inventory.Site{Metadata: inventory.Metadata{Name: "Temporary"}})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestSiteRepositoryDeleteNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestSiteRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	fixedID := uuid.New()
	repo, ctx := newTestRepository(t, id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, inventory.Site{Metadata: inventory.Metadata{Name: "First"}}); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, inventory.Site{Metadata: inventory.Metadata{Name: "Second"}})
	if err == nil {
		t.Fatal("second Create() = nil, want a conflict error for the duplicate primary key")
	}
	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindConflict, err)
	}
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want a not-found error")
	}
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindNotFound, err)
	}
}

// testConfig points at the same database.Config the rest of the test suite
// uses (see internal/database/integration_test.go): local defaults that
// match docker-compose.yml, overridable via environment variables.
func testConfig(t *testing.T) database.Config {
	t.Helper()
	return database.Config{
		Host:            envOrDefault("DB_HOST", "localhost"),
		Port:            5432,
		User:            envOrDefault("DB_USER", "palladium"),
		Password:        envOrDefault("DB_PASSWORD", "palladium"),
		Database:        envOrDefault("DB_NAME", "palladium"),
		SSLMode:         "disable",
		MaxConns:        2,
		MinConns:        1,
		MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute,
		ConnectTimeout:  5 * time.Second,
	}
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
