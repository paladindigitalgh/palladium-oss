//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestRepository mirrors internal/inventory/postgres/site_test.go's
// helper of the same name: open a transaction against the real test
// database, build the repository under test on it, and roll the
// transaction back on cleanup so tests never leave data behind or observe
// each other's writes.
func newTestRepository(t *testing.T, ids id.Generator) (*postgres.UserRepository, context.Context) {
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

	return postgres.NewUserRepository(tx, clock.New(), ids), ctx
}

func testUser(email string) auth.User {
	return auth.User{Email: email, PasswordHash: "$2a$10$examplehashexamplehashexampleu"}
}

func TestUserRepositoryCount(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() = %v", err)
	}
	if count != 0 {
		t.Fatalf("Count() = %d, want 0 on a fresh transaction", count)
	}

	if _, err := repo.Create(ctx, testUser("first@example.com")); err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if _, err := repo.Create(ctx, testUser("second@example.com")); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() = %v", err)
	}
	if count != 2 {
		t.Fatalf("Count() = %d, want 2 after creating two users", count)
	}
}

func TestUserRepositoryCreate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testUser("jane@example.com"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.Email != "jane@example.com" {
		t.Errorf("Email = %q, want %q", created.Email, "jane@example.com")
	}
	if created.PasswordHash != "$2a$10$examplehashexamplehashexampleu" {
		t.Errorf("PasswordHash = %q, want the input hash unchanged", created.PasswordHash)
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestUserRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	user := testUser("edge@example.com")
	user.ID = bogusID
	user.CreatedAt = bogusTime
	user.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, user)
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

func TestUserRepositoryCreateConflictOnDuplicateEmail(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	if _, err := repo.Create(ctx, testUser("duplicate@example.com")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testUser("duplicate@example.com"))
	if err == nil {
		t.Fatal("second Create() = nil, want a conflict error for the duplicate email")
	}
	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindConflict, err)
	}
}

func TestUserRepositoryGetByID(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testUser("jane@example.com"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() = %v", err)
	}
	if got.ID != created.ID || got.Email != created.Email {
		t.Errorf("GetByID() = %+v, want %+v", got, created)
	}
}

func TestUserRepositoryGetByIDNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.GetByID(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestUserRepositoryGetByEmail(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testUser("jane@example.com"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.GetByEmail(ctx, "jane@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetByEmail() ID = %v, want %v", got.ID, created.ID)
	}
}

func TestUserRepositoryGetByEmailNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.GetByEmail(ctx, "nobody@example.com")

	assertNotFound(t, err)
}

func TestUserRepositoryUpdatePasswordHash(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testUser("jane@example.com"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.UpdatePasswordHash(ctx, created.ID, "$2a$10$anewhashanewhashanewhashanewhu")
	if err != nil {
		t.Fatalf("UpdatePasswordHash() = %v", err)
	}

	if updated.PasswordHash != "$2a$10$anewhashanewhashanewhashanewhu" {
		t.Errorf("PasswordHash = %q, want the new hash", updated.PasswordHash)
	}
	if updated.Email != created.Email {
		t.Errorf("Email changed: was %q, now %q", created.Email, updated.Email)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on UpdatePasswordHash(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestUserRepositoryUpdatePasswordHashNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.UpdatePasswordHash(ctx, uuid.New(), "$2a$10$anewhashanewhashanewhashanewhu")

	assertNotFound(t, err)
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
// uses (see internal/database/integration_test.go and
// internal/inventory/postgres/testing_test.go): local defaults that match
// docker-compose.yml, overridable via environment variables.
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
