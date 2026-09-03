//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/event/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestRepository opens a transaction against the real test database
// and builds an EventRepository on top of it, rolled back automatically
// on cleanup — the same shape internal/inventory/postgres's own
// newTestRepository/newTestQuerier helpers use, duplicated here rather
// than shared across packages (see that package's own comment on why:
// each entity's test file owns its setup).
func newTestRepository(t *testing.T) (*postgres.EventRepository, context.Context) {
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

	return postgres.NewEventRepository(tx, clock.New(), id.New()), ctx
}

// testConfig mirrors internal/inventory/postgres/site_test.go's own
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

func createTestEvent(t *testing.T, ctx context.Context, repo *postgres.EventRepository, message string) event.Event {
	t.Helper()

	e, err := repo.Create(ctx, event.Event{
		EntityType: "service",
		EntityID:   uuid.New(),
		Type:       "workflow.started",
		Message:    message,
	})
	if err != nil {
		t.Fatalf("fixture: create event: %v", err)
	}
	return e
}

func TestEventRepositoryListRecentOrdersNewestFirst(t *testing.T) {
	repo, ctx := newTestRepository(t)

	first := createTestEvent(t, ctx, repo, "first")
	second := createTestEvent(t, ctx, repo, "second")
	third := createTestEvent(t, ctx, repo, "third")

	events, err := repo.ListRecent(ctx, 10)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}

	if len(events) < 3 {
		t.Fatalf("len(events) = %d, want at least 3", len(events))
	}

	// The three just-created events must appear in this transaction's
	// view newest-first, relative to each other -- other tests running
	// concurrently against the same database may add more rows, but
	// they cannot see this transaction's uncommitted ones, so `events`
	// here consists exactly of what this test itself created.
	ids := make([]uuid.UUID, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	wantOrder := []uuid.UUID{third.ID, second.ID, first.ID}
	for i, want := range wantOrder {
		if ids[i] != want {
			t.Errorf("events[%d].ID = %v, want %v (newest first)", i, ids[i], want)
		}
	}
}

func TestEventRepositoryListRecentRespectsLimit(t *testing.T) {
	repo, ctx := newTestRepository(t)

	for i := 0; i < 5; i++ {
		createTestEvent(t, ctx, repo, "event")
	}

	events, err := repo.ListRecent(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
}
