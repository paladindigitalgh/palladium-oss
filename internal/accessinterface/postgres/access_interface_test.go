//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	accessnetworkpostgres "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	oltpostgres "github.com/paladindigitalgh/palladium-oss/internal/olt/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
	ponportpostgres "github.com/paladindigitalgh/palladium-oss/internal/ponport/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/ponport/postgres/pon_port_test.go. Every AccessInterface test
// needs a fixture PONPort (which itself needs a fixture OLT, which needs
// a fixture AccessNetwork) to satisfy the required PONPortID foreign
// key, and the fixture must share the same transaction as the repository
// under test, so tests here call this directly rather than hiding it
// behind an AccessInterface-style newTestRepository wrapper.
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

// createTestPONPort creates a real PONPort row (and the fixture OLT and
// AccessNetwork it requires) through internal/ponport/postgres,
// internal/olt/postgres, and internal/accessnetwork/postgres — not
// internal/accessinterface/postgres — so an AccessInterface fixture
// failure surfaces as a clear failure of one specific layer, not a
// confusing failure somewhere else. This is the one place this package
// imports internal/ponport, internal/olt, or internal/accessnetwork at
// all: the domain model (internal/accessinterface) never does (see its
// package doc comment), only this test, which genuinely needs a real
// pon_ports row for the foreign key to reference.
func createTestPONPort(t *testing.T, ctx context.Context, q database.Querier) ponport.PONPort {
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

	ponPortRepo := ponportpostgres.NewPONPortRepository(q, clock.New(), id.New())
	p, err := ponPortRepo.Create(ctx, ponport.PONPort{
		OLTID:      o.ID,
		PortNumber: 1,
	})
	if err != nil {
		t.Fatalf("fixture: create pon port: %v", err)
	}
	return p
}

func testAccessInterface(ponPortID uuid.UUID, name string) accessinterface.AccessInterface {
	return accessinterface.AccessInterface{
		PONPortID:  ponPortID,
		Technology: accessinterface.TechnologyGPON,
		Name:       name,
		Status:     accessinterface.StatusActive,
	}
}

func TestAccessInterfaceRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, accessinterface.AccessInterface{
		PONPortID:   p.ID,
		Technology:  accessinterface.TechnologyGPON,
		Name:        "gpon-0/1/1",
		Status:      accessinterface.StatusActive,
		Description: "North cabinet drop",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.PONPortID != p.ID {
		t.Errorf("PONPortID = %v, want %v", created.PONPortID, p.ID)
	}
	if created.Technology != accessinterface.TechnologyGPON {
		t.Errorf("Technology = %q, want %q", created.Technology, accessinterface.TechnologyGPON)
	}
	if created.Name != "gpon-0/1/1" {
		t.Errorf("Name = %q, want %q", created.Name, "gpon-0/1/1")
	}
	if created.Status != accessinterface.StatusActive {
		t.Errorf("Status = %q, want %q", created.Status, accessinterface.StatusActive)
	}
	if created.Description != "North cabinet drop" {
		t.Errorf("Description = %q, want %q", created.Description, "North cabinet drop")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestAccessInterfaceRepositoryCreatePersistsEachDefinedTechnologyAndStatus(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	technologies := []accessinterface.Technology{
		accessinterface.TechnologyGPON,
		accessinterface.TechnologyXGSPON,
		accessinterface.TechnologyActiveEthernet,
		accessinterface.TechnologyOther,
	}
	for _, tech := range technologies {
		a := testAccessInterface(p.ID, uuid.NewString())
		a.Technology = tech

		created, err := repo.Create(ctx, a)
		if err != nil {
			t.Fatalf("Create() (technology %q) = %v", tech, err)
		}
		if created.Technology != tech {
			t.Errorf("Technology = %q, want %q", created.Technology, tech)
		}
	}

	statuses := []accessinterface.Status{
		accessinterface.StatusActive,
		accessinterface.StatusDisabled,
	}
	for _, s := range statuses {
		a := testAccessInterface(p.ID, uuid.NewString())
		a.Status = s

		created, err := repo.Create(ctx, a)
		if err != nil {
			t.Fatalf("Create() (status %q) = %v", s, err)
		}
		if created.Status != s {
			t.Errorf("Status = %q, want %q", created.Status, s)
		}
	}
}

func TestAccessInterfaceRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	a := testAccessInterface(p.ID, "Edge Interface")
	a.ID = bogusID
	a.CreatedAt = bogusTime
	a.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, a)
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

func TestAccessInterfaceRepositoryCreateFailsWhenPONPortDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testAccessInterface(uuid.New(), "Orphan Interface")) // pon port does not exist

	assertConflict(t, err)
}

func TestAccessInterfaceRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessInterface(p.ID, "Interface-Get"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.PONPortID != created.PONPortID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestAccessInterfaceRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestAccessInterfaceRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testAccessInterface(p.ID, "Alpha Interface"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testAccessInterface(p.ID, "Beta Interface"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	interfaces, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]accessinterface.AccessInterface, len(interfaces))
	for _, a := range interfaces {
		found[a.ID] = a
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created AccessInterface")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created AccessInterface")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture PONPort/OLT/AccessNetwork, different tables), so the
	// list is exactly these two, letting us also check the ORDER BY name.
	if len(interfaces) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(interfaces), interfaces)
	}
	if interfaces[0].Name != "Alpha Interface" || interfaces[1].Name != "Beta Interface" {
		t.Errorf("List() order = [%q, %q], want [Alpha Interface, Beta Interface]", interfaces[0].Name, interfaces[1].Name)
	}
}

func TestAccessInterfaceRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	otherPONPort := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessInterface(p.ID, "Old Name"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, accessinterface.AccessInterface{
		ID:          created.ID,
		PONPortID:   otherPONPort.ID,
		Technology:  accessinterface.TechnologyXGSPON,
		Name:        "New Name",
		Status:      accessinterface.StatusDisabled,
		Description: "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.PONPortID != otherPONPort.ID {
		t.Errorf("PONPortID = %v, want %v (PONPortID must be mutable via Update)", updated.PONPortID, otherPONPort.ID)
	}
	if updated.Technology != accessinterface.TechnologyXGSPON {
		t.Errorf("Technology = %q, want %q", updated.Technology, accessinterface.TechnologyXGSPON)
	}
	if updated.Status != accessinterface.StatusDisabled {
		t.Errorf("Status = %q, want %q", updated.Status, accessinterface.StatusDisabled)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestAccessInterfaceRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	ghost := testAccessInterface(p.ID, "Ghost")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestAccessInterfaceRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessInterface(p.ID, "Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestAccessInterfaceRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestAccessInterfaceRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testAccessInterface(p.ID, "First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testAccessInterface(p.ID, "Second"))
	assertConflict(t, err)
}

// TestPONPortRepositoryDeleteBlockedByExistingAccessInterface lives
// here, not in internal/ponport/postgres, so that package's existing
// test files stay untouched — the same reasoning
// internal/olt/postgres/olt_test.go already documents for why its
// equivalent test lives with the child, not the parent. It exercises
// PONPortRepository.Delete against the foreign key this migration adds.
func TestPONPortRepositoryDeleteBlockedByExistingAccessInterface(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestPONPort(t, ctx, q)
	interfaceRepo := postgres.NewAccessInterfaceRepository(q, clock.New(), id.New())
	if _, err := interfaceRepo.Create(ctx, testAccessInterface(p.ID, "Blocking Interface")); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	ponPortRepo := ponportpostgres.NewPONPortRepository(q, clock.New(), id.New())

	err := ponPortRepo.Delete(ctx, p.ID)

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
