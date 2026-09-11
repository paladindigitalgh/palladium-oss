//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authpostgres "github.com/paladindigitalgh/palladium-oss/internal/auth/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/note"
	"github.com/paladindigitalgh/palladium-oss/internal/note/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as every other
// repository test file in this codebase (see e.g.
// internal/contact/postgres/contact_test.go). Every Note test needs a
// fixture User to satisfy the required AuthorUserID foreign key, and the
// fixture must share the same transaction as the repository under test,
// so tests here call createTestUser directly rather than hiding it
// behind a wrapper.
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

// testConfig mirrors internal/contact/postgres/contact_test.go's own
// testConfig exactly: local defaults matching docker-compose.yml,
// overridable via environment variables.
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
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// createTestUser creates a real User row through internal/auth/postgres —
// not internal/note/postgres — so a Note fixture failure surfaces as a
// clear failure of User's own Create, not a confusing failure somewhere
// else. This is the one place this test package touches internal/auth at
// all: the domain model (internal/note) never does (see its package doc
// comment), only this test, which genuinely needs a real users row for
// the foreign key to reference.
func createTestUser(t *testing.T, ctx context.Context, q database.Querier) auth.User {
	t.Helper()

	repo := authpostgres.NewUserRepository(q, clock.New(), id.New())
	u, err := repo.Create(ctx, auth.User{
		Email:        "operator-" + uuid.NewString() + "@example.com",
		PasswordHash: "fixture-hash",
		FirstName:    "Jane",
		LastName:     "Doe",
		Role:         auth.RoleOperator,
		Status:       auth.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create user: %v", err)
	}
	return u
}

func TestNoteRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	u := createTestUser(t, ctx, q)
	repo := postgres.NewNoteRepository(q, clock.New(), id.New())

	entityID := uuid.New()
	created, err := repo.Create(ctx, note.Note{
		EntityType:      "customer",
		EntityID:        entityID,
		AuthorUserID:    u.ID,
		AuthorEmail:     u.Email,
		AuthorFirstName: u.FirstName,
		AuthorLastName:  u.LastName,
		Body:            "Called the customer back, issue resolved.",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.EntityType != "customer" || created.EntityID != entityID {
		t.Errorf("entity = %s/%v, want customer/%v", created.EntityType, created.EntityID, entityID)
	}
	if created.AuthorUserID != u.ID || created.AuthorEmail != u.Email {
		t.Errorf("author = %v/%q, want %v/%q", created.AuthorUserID, created.AuthorEmail, u.ID, u.Email)
	}
	if created.AuthorFirstName != "Jane" || created.AuthorLastName != "Doe" {
		t.Errorf("author name = %q %q, want Jane Doe", created.AuthorFirstName, created.AuthorLastName)
	}
	if created.Body != "Called the customer back, issue resolved." {
		t.Errorf("Body = %q, want %q", created.Body, "Called the customer back, issue resolved.")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
}

func TestNoteRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	u := createTestUser(t, ctx, q)
	repo := postgres.NewNoteRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	created, err := repo.Create(ctx, note.Note{
		ID:           bogusID,
		EntityType:   "device",
		EntityID:     uuid.New(),
		AuthorUserID: u.ID,
		AuthorEmail:  u.Email,
		Body:         "Edge note",
		CreatedAt:    bogusTime,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == bogusID {
		t.Error("Create() used the caller-supplied ID instead of generating one")
	}
	if created.CreatedAt.Equal(bogusTime) {
		t.Error("Create() used the caller-supplied CreatedAt instead of stamping now")
	}
}

func TestNoteRepositoryCreateRejectsUnknownAuthor(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewNoteRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, note.Note{
		EntityType:   "service",
		EntityID:     uuid.New(),
		AuthorUserID: uuid.New(), // no such user exists
		AuthorEmail:  "ghost@example.com",
		Body:         "Should fail",
	})
	if err == nil {
		t.Fatal("Create() = nil, want a foreign key conflict error")
	}
}

func TestNoteRepositoryListByEntity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	u := createTestUser(t, ctx, q)
	repo := postgres.NewNoteRepository(q, clock.New(), id.New())

	entityID := uuid.New()
	other := uuid.New()

	first, err := repo.Create(ctx, note.Note{
		EntityType: "customer", EntityID: entityID, AuthorUserID: u.ID, AuthorEmail: u.Email, Body: "First note",
	})
	if err != nil {
		t.Fatalf("fixture: create first note: %v", err)
	}
	second, err := repo.Create(ctx, note.Note{
		EntityType: "customer", EntityID: entityID, AuthorUserID: u.ID, AuthorEmail: u.Email, Body: "Second note",
	})
	if err != nil {
		t.Fatalf("fixture: create second note: %v", err)
	}
	if _, err := repo.Create(ctx, note.Note{
		EntityType: "customer", EntityID: other, AuthorUserID: u.ID, AuthorEmail: u.Email, Body: "Different entity",
	}); err != nil {
		t.Fatalf("fixture: create unrelated note: %v", err)
	}

	notes, err := repo.ListByEntity(ctx, "customer", entityID)
	if err != nil {
		t.Fatalf("ListByEntity() = %v", err)
	}

	if len(notes) != 2 {
		t.Fatalf("len(notes) = %d, want 2", len(notes))
	}
	if notes[0].ID != second.ID || notes[1].ID != first.ID {
		t.Errorf("notes = [%q, %q], want newest first: [%q, %q]", notes[0].Body, notes[1].Body, second.Body, first.Body)
	}
}

func TestNoteRepositoryListByEntityReturnsEmptyNotNilForUnknownEntity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewNoteRepository(q, clock.New(), id.New())

	notes, err := repo.ListByEntity(ctx, "customer", uuid.New())
	if err != nil {
		t.Fatalf("ListByEntity() = %v", err)
	}
	if notes == nil {
		t.Error("ListByEntity() = nil, want an empty slice")
	}
	if len(notes) != 0 {
		t.Errorf("len(notes) = %d, want 0", len(notes))
	}
}
