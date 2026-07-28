//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	"github.com/paladindigitalgh/palladium-oss/internal/catalog/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestRepository mirrors internal/customer/postgres/customer_test.go's
// helper of the same name: open a transaction against the real test
// database, build the repository under test on it, and roll the
// transaction back on cleanup so tests never leave data behind or observe
// each other's writes.
func newTestRepository(t *testing.T, ids id.Generator) (*postgres.CatalogRepository, context.Context) {
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

	return postgres.NewCatalogRepository(tx, clock.New(), ids), ctx
}

func testCatalog(name string) catalog.ProductCatalog {
	return catalog.ProductCatalog{
		Name:   name,
		Status: catalog.CatalogStatusActive,
	}
}

func TestCatalogRepositoryCreate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, catalog.ProductCatalog{
		Name:        "Residential",
		Status:      catalog.CatalogStatusActive,
		Description: "Products sold to residential customers",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.Name != "Residential" {
		t.Errorf("Name = %q, want %q", created.Name, "Residential")
	}
	if created.Status != catalog.CatalogStatusActive {
		t.Errorf("Status = %q, want %q", created.Status, catalog.CatalogStatusActive)
	}
	if created.Description != "Products sold to residential customers" {
		t.Errorf("Description = %q, want %q", created.Description, "Products sold to residential customers")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestCatalogRepositoryCreatePersistsEachDefinedStatus(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	statuses := []catalog.CatalogStatus{
		catalog.CatalogStatusActive,
		catalog.CatalogStatusInactive,
	}

	for _, st := range statuses {
		c := testCatalog(uuid.NewString())
		c.Status = st

		created, err := repo.Create(ctx, c)
		if err != nil {
			t.Fatalf("Create() (status %q) = %v", st, err)
		}
		if created.Status != st {
			t.Errorf("Status = %q, want %q", created.Status, st)
		}

		got, err := repo.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Get() = %v", err)
		}
		if got.Status != st {
			t.Errorf("Get() Status = %q, want %q", got.Status, st)
		}
	}
}

func TestCatalogRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	c := testCatalog("Edge Catalog")
	c.ID = bogusID
	c.CreatedAt = bogusTime
	c.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, c)
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

func TestCatalogRepositoryGet(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testCatalog("Business"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name || got.Status != created.Status {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestCatalogRepositoryGetNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestCatalogRepositoryList(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	first, err := repo.Create(ctx, testCatalog("Alpha Catalog"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testCatalog("Beta Catalog"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	catalogs, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]catalog.ProductCatalog, len(catalogs))
	for _, c := range catalogs {
		found[c.ID] = c
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created catalog")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created catalog")
	}

	// Both were created within this same rolled-back transaction, so the
	// list is exactly these two, letting us also check the ORDER BY name.
	if len(catalogs) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(catalogs), catalogs)
	}
	if catalogs[0].Name != "Alpha Catalog" || catalogs[1].Name != "Beta Catalog" {
		t.Errorf("List() order = [%q, %q], want [Alpha Catalog, Beta Catalog]", catalogs[0].Name, catalogs[1].Name)
	}
}

func TestCatalogRepositoryUpdate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, catalog.ProductCatalog{
		Name:        "Old Name",
		Status:      catalog.CatalogStatusActive,
		Description: "Old Description",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, catalog.ProductCatalog{
		ID:          created.ID,
		Name:        "New Name",
		Status:      catalog.CatalogStatusInactive,
		Description: "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != catalog.CatalogStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, catalog.CatalogStatusInactive)
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

func TestCatalogRepositoryUpdateNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Update(ctx, catalog.ProductCatalog{
		ID:     uuid.New(),
		Name:   "Ghost",
		Status: catalog.CatalogStatusActive,
	})

	assertNotFound(t, err)
}

func TestCatalogRepositoryDelete(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testCatalog("Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestCatalogRepositoryDeleteNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestCatalogRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	fixedID := uuid.New()
	repo, ctx := newTestRepository(t, id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testCatalog("First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testCatalog("Second"))
	assertConflict(t, err)
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

func assertConflict(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want a conflict error")
	}
	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindConflict, err)
	}
}

// testConfig points at the same database.Config the rest of the test
// suite uses (see internal/customer/postgres/customer_test.go): local
// defaults that match docker-compose.yml, overridable via environment
// variables.
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
