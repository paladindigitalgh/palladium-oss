//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	accessnetworkpostgres "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	oltpostgres "github.com/paladindigitalgh/palladium-oss/internal/olt/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/olt/postgres/olt_test.go. Every PONPort test needs a fixture
// OLT (which itself needs a fixture AccessNetwork) to satisfy the
// required OLTID foreign key, and the fixture must share the same
// transaction as the repository under test, so tests here call this
// directly rather than hiding it behind an OLT-style newTestRepository
// wrapper.
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

// createTestOLT creates a real OLT row (and the fixture AccessNetwork it
// requires) through internal/olt/postgres and
// internal/accessnetwork/postgres — not internal/ponport/postgres — so a
// PONPort fixture failure surfaces as a clear failure of one specific
// layer, not a confusing failure somewhere else. This is the one place
// this package imports internal/olt or internal/accessnetwork at all:
// the domain model (internal/ponport) never does (see its package doc
// comment), only this test, which genuinely needs a real olts row for
// the foreign key to reference.
func createTestOLT(t *testing.T, ctx context.Context, q database.Querier) olt.OLT {
	t.Helper()

	accessNetworkRepo := accessnetworkpostgres.NewAccessNetworkRepository(q, clock.New(), id.New())
	a, err := accessNetworkRepo.Create(ctx, accessnetwork.AccessNetwork{
		Name:   "Fixture Access Network " + uuid.NewString(),
		Status: accessnetwork.AccessNetworkStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create access network: %v", err)
	}

	oltRepo := oltpostgres.NewOLTRepository(q, clock.New(), id.New())
	o, err := oltRepo.Create(ctx, olt.OLT{
		AccessNetworkID: a.ID,
		Name:            "Fixture OLT " + uuid.NewString(),
		Vendor:          olt.VendorKontron,
	})
	if err != nil {
		t.Fatalf("fixture: create olt: %v", err)
	}
	return o
}

func testPONPort(oltID uuid.UUID, portNumber int) ponport.PONPort {
	return ponport.PONPort{
		OLTID:      oltID,
		PortNumber: portNumber,
	}
}

func TestPONPortRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, ponport.PONPort{
		OLTID:       o.ID,
		PortNumber:  1,
		Description: "Feeds the north neighborhood splitter",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.OLTID != o.ID {
		t.Errorf("OLTID = %v, want %v", created.OLTID, o.ID)
	}
	if created.PortNumber != 1 {
		t.Errorf("PortNumber = %d, want %d", created.PortNumber, 1)
	}
	if created.Description != "Feeds the north neighborhood splitter" {
		t.Errorf("Description = %q, want %q", created.Description, "Feeds the north neighborhood splitter")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestPONPortRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	p := testPONPort(o.ID, 1)
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

func TestPONPortRepositoryCreateFailsWhenOLTDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testPONPort(uuid.New(), 1)) // olt does not exist

	assertConflict(t, err)
}

func TestPONPortRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testPONPort(o.ID, 1))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.OLTID != created.OLTID || got.PortNumber != created.PortNumber {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestPONPortRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestPONPortRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	second, err := repo.Create(ctx, testPONPort(o.ID, 2))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	first, err := repo.Create(ctx, testPONPort(o.ID, 1))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	ports, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]ponport.PONPort, len(ports))
	for _, p := range ports {
		found[p.ID] = p
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created port")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created port")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture chain, all different tables), so the list is exactly
	// these two, letting us also check the ORDER BY port_number —
	// created in the order 2, then 1, so List() must still come back
	// [1, 2].
	if len(ports) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(ports), ports)
	}
	if ports[0].PortNumber != 1 || ports[1].PortNumber != 2 {
		t.Errorf("List() order = [%d, %d], want [1, 2]", ports[0].PortNumber, ports[1].PortNumber)
	}
}

func TestPONPortRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	otherOLT := createTestOLT(t, ctx, q)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testPONPort(o.ID, 1))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, ponport.PONPort{
		ID:          created.ID,
		OLTID:       otherOLT.ID,
		PortNumber:  2,
		Description: "Moved to a different OLT",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.OLTID != otherOLT.ID {
		t.Errorf("OLTID = %v, want %v (OLTID must be mutable via Update)", updated.OLTID, otherOLT.ID)
	}
	if updated.PortNumber != 2 {
		t.Errorf("PortNumber = %d, want %d", updated.PortNumber, 2)
	}
	if updated.Description != "Moved to a different OLT" {
		t.Errorf("Description = %q, want %q", updated.Description, "Moved to a different OLT")
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestPONPortRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	ghost := testPONPort(o.ID, 1)
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestPONPortRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testPONPort(o.ID, 1))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestPONPortRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewPONPortRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestPONPortRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewPONPortRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testPONPort(o.ID, 1)); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testPONPort(o.ID, 2))
	assertConflict(t, err)
}

// TestOLTRepositoryDeleteBlockedByExistingPONPort lives here, not in
// internal/olt/postgres, so that package's existing test files stay
// untouched — the same reasoning
// internal/olt/postgres/olt_test.go already documents for why its
// equivalent test (blocking an AccessNetwork delete) lives with the
// child, not the parent. It exercises OLTRepository.Delete against the
// foreign key this migration adds.
func TestOLTRepositoryDeleteBlockedByExistingPONPort(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	portRepo := postgres.NewPONPortRepository(q, clock.New(), id.New())
	if _, err := portRepo.Create(ctx, testPONPort(o.ID, 1)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	oltRepo := oltpostgres.NewOLTRepository(q, clock.New(), id.New())

	err := oltRepo.Delete(ctx, o.ID)

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
