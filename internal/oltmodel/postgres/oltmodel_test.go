//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestRepository mirrors internal/provider/postgres/provider_test.go's
// helper of the same name: open a transaction against the real test
// database, build the repository under test on it, and roll the
// transaction back on cleanup so tests never leave data behind or observe
// each other's writes.
func newTestRepository(t *testing.T, ids id.Generator) (*postgres.OLTModelRepository, context.Context) {
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

	return postgres.NewOLTModelRepository(tx, clock.New(), ids), ctx
}

func testOLTModel(name string) oltmodel.OLTModel {
	return oltmodel.OLTModel{
		Vendor:       oltmodel.VendorKontron,
		Name:         name,
		PONPortCount: 16,
	}
}

func TestOLTModelRepositoryCreate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, oltmodel.OLTModel{
		Vendor:       oltmodel.VendorKontron,
		Name:         "C16",
		PONPortCount: 16,
		Description:  "16-port GPON OLT chassis",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.Vendor != oltmodel.VendorKontron {
		t.Errorf("Vendor = %q, want %q", created.Vendor, oltmodel.VendorKontron)
	}
	if created.Name != "C16" {
		t.Errorf("Name = %q, want %q", created.Name, "C16")
	}
	if created.PONPortCount != 16 {
		t.Errorf("PONPortCount = %d, want %d", created.PONPortCount, 16)
	}
	if created.Description != "16-port GPON OLT chassis" {
		t.Errorf("Description = %q, want %q", created.Description, "16-port GPON OLT chassis")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestOLTModelRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	m := testOLTModel("Edge Model")
	m.ID = bogusID
	m.CreatedAt = bogusTime
	m.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, m)
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

func TestOLTModelRepositoryGet(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testOLTModel("C8"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name || got.Vendor != created.Vendor {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestOLTModelRepositoryGetNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestOLTModelRepositoryList(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	first, err := repo.Create(ctx, testOLTModel("Alpha16"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testOLTModel("Beta32"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	models, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]oltmodel.OLTModel, len(models))
	for _, m := range models {
		found[m.ID] = m
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created olt model")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created olt model")
	}

	// Both were created within this same rolled-back transaction, so the
	// list is exactly these two, letting us also check the ORDER BY name.
	if len(models) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(models), models)
	}
	if models[0].Name != "Alpha16" || models[1].Name != "Beta32" {
		t.Errorf("List() order = [%q, %q], want [Alpha16, Beta32]", models[0].Name, models[1].Name)
	}
}

func TestOLTModelRepositoryUpdate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, oltmodel.OLTModel{
		Vendor:       oltmodel.VendorKontron,
		Name:         "Old Name",
		PONPortCount: 8,
		Description:  "Old Description",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, oltmodel.OLTModel{
		ID:           created.ID,
		Vendor:       oltmodel.VendorNokia,
		Name:         "New Name",
		PONPortCount: 32,
		Description:  "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Vendor != oltmodel.VendorNokia {
		t.Errorf("Vendor = %q, want %q", updated.Vendor, oltmodel.VendorNokia)
	}
	if updated.PONPortCount != 32 {
		t.Errorf("PONPortCount = %d, want %d", updated.PONPortCount, 32)
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

func TestOLTModelRepositoryUpdateNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Update(ctx, oltmodel.OLTModel{
		ID:           uuid.New(),
		Vendor:       oltmodel.VendorKontron,
		Name:         "Ghost",
		PONPortCount: 16,
	})

	assertNotFound(t, err)
}

func TestOLTModelRepositoryDelete(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testOLTModel("Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestOLTModelRepositoryDeleteNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestOLTModelRepositoryCreateConflictOnDuplicateVendorAndName(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	if _, err := repo.Create(ctx, testOLTModel("C16")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testOLTModel("C16"))
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
