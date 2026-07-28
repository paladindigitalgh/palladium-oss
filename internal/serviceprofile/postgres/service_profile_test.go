//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/postgres"
)

// newTestRepository mirrors internal/catalog/postgres/catalog_test.go's
// helper of the same name: open a transaction against the real test
// database, build the repository under test on it, and roll the
// transaction back on cleanup so tests never leave data behind or observe
// each other's writes.
func newTestRepository(t *testing.T, ids id.Generator) (*postgres.ServiceProfileRepository, context.Context) {
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

	return postgres.NewServiceProfileRepository(tx, clock.New(), ids), ctx
}

func testServiceProfile(name string) serviceprofile.ServiceProfile {
	return serviceprofile.ServiceProfile{
		Name:   name,
		Status: serviceprofile.StatusActive,
	}
}

func TestServiceProfileRepositoryCreate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, serviceprofile.ServiceProfile{
		Name:        "Residential Internet",
		Status:      serviceprofile.StatusActive,
		Description: "Standard residential internet service",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.Name != "Residential Internet" {
		t.Errorf("Name = %q, want %q", created.Name, "Residential Internet")
	}
	if created.Status != serviceprofile.StatusActive {
		t.Errorf("Status = %q, want %q", created.Status, serviceprofile.StatusActive)
	}
	if created.Description != "Standard residential internet service" {
		t.Errorf("Description = %q, want %q", created.Description, "Standard residential internet service")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestServiceProfileRepositoryCreatePersistsEachDefinedStatus(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	statuses := []serviceprofile.Status{
		serviceprofile.StatusActive,
		serviceprofile.StatusInactive,
	}

	for _, st := range statuses {
		p := testServiceProfile(uuid.NewString())
		p.Status = st

		created, err := repo.Create(ctx, p)
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

func TestServiceProfileRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	p := testServiceProfile("Edge Profile")
	p.ID = bogusID
	p.CreatedAt = bogusTime
	p.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, p)
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

func TestServiceProfileRepositoryGet(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testServiceProfile("Business Ethernet"))
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

func TestServiceProfileRepositoryGetNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestServiceProfileRepositoryList(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	first, err := repo.Create(ctx, testServiceProfile("Alpha Profile"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testServiceProfile("Beta Profile"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	profiles, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]serviceprofile.ServiceProfile, len(profiles))
	for _, p := range profiles {
		found[p.ID] = p
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created service profile")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created service profile")
	}

	// Both were created within this same rolled-back transaction, so the
	// list is exactly these two, letting us also check the ORDER BY name.
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(profiles), profiles)
	}
	if profiles[0].Name != "Alpha Profile" || profiles[1].Name != "Beta Profile" {
		t.Errorf("List() order = [%q, %q], want [Alpha Profile, Beta Profile]", profiles[0].Name, profiles[1].Name)
	}
}

func TestServiceProfileRepositoryUpdate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, serviceprofile.ServiceProfile{
		Name:        "Old Name",
		Status:      serviceprofile.StatusActive,
		Description: "Old Description",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, serviceprofile.ServiceProfile{
		ID:          created.ID,
		Name:        "New Name",
		Status:      serviceprofile.StatusInactive,
		Description: "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != serviceprofile.StatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, serviceprofile.StatusInactive)
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

func TestServiceProfileRepositoryUpdateNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Update(ctx, serviceprofile.ServiceProfile{
		ID:     uuid.New(),
		Name:   "Ghost",
		Status: serviceprofile.StatusActive,
	})

	assertNotFound(t, err)
}

func TestServiceProfileRepositoryDelete(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testServiceProfile("Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestServiceProfileRepositoryDeleteNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestServiceProfileRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	fixedID := uuid.New()
	repo, ctx := newTestRepository(t, id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testServiceProfile("First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testServiceProfile("Second"))
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
