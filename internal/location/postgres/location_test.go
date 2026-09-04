//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as every other
// repository test file in this codebase (see e.g.
// internal/inventory/postgres/testing_test.go). Every Location test needs
// a fixture Customer to satisfy the required CustomerID foreign key, and
// the fixture must share the same transaction as the repository under
// test, so tests here call this directly rather than hiding it behind a
// Site-style newTestRepository wrapper.
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

// createTestCustomer creates a real Customer row through
// internal/customer/postgres — not internal/location/postgres — so a
// Location fixture failure surfaces as a clear failure of Customer's own
// Create, not a confusing failure somewhere else. This is the one place
// this package imports internal/customer at all: the domain model
// (internal/location) never does (see its package doc comment), only this
// test, which genuinely needs a real customers row for the foreign key to
// reference.
func createTestCustomer(t *testing.T, ctx context.Context, q database.Querier) customer.Customer {
	t.Helper()

	repo := customerpostgres.NewCustomerRepository(q, clock.New(), id.New())
	c, err := repo.Create(ctx, customer.Customer{
		Name:         "Fixture Customer " + uuid.NewString(),
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create customer: %v", err)
	}
	return c
}

func testLocation(customerID uuid.UUID, name string) location.Location {
	return location.Location{
		CustomerID: customerID,
		Name:       name,
		Type:       location.LocationTypeService,
		Status:     location.LocationStatusActive,
	}
}

func TestLocationRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	lat, lng := 39.7817, -89.6501
	created, err := repo.Create(ctx, location.Location{
		CustomerID:  c.ID,
		Name:        "Main Service Address",
		Type:        location.LocationTypeService,
		Status:      location.LocationStatusActive,
		Address1:    "123 Main St",
		City:        "Springfield",
		State:       "IL",
		PostalCode:  "62701",
		Country:     "US",
		Latitude:    &lat,
		Longitude:   &lng,
		Description: "Primary residence",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.CustomerID != c.ID {
		t.Errorf("CustomerID = %v, want %v", created.CustomerID, c.ID)
	}
	if created.Name != "Main Service Address" {
		t.Errorf("Name = %q, want %q", created.Name, "Main Service Address")
	}
	if created.Type != location.LocationTypeService {
		t.Errorf("Type = %q, want %q", created.Type, location.LocationTypeService)
	}
	if created.Status != location.LocationStatusActive {
		t.Errorf("Status = %q, want %q", created.Status, location.LocationStatusActive)
	}
	if created.Address1 != "123 Main St" || created.City != "Springfield" {
		t.Errorf("address fields = %+v, want Address1=123 Main St City=Springfield", created)
	}
	if created.Latitude == nil || *created.Latitude != lat {
		t.Errorf("Latitude = %v, want %v", created.Latitude, lat)
	}
	if created.Longitude == nil || *created.Longitude != lng {
		t.Errorf("Longitude = %v, want %v", created.Longitude, lng)
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

// TestLocationRepositoryCreateWithoutCoordinates proves Latitude/Longitude
// round-trip as nil, not silently becoming 0 — the exact distinction
// *float64 exists to preserve (see location.Location's doc comment).
func TestLocationRepositoryCreateWithoutCoordinates(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testLocation(c.ID, "No Coordinates"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.Latitude != nil {
		t.Errorf("Latitude = %v, want nil", *created.Latitude)
	}
	if created.Longitude != nil {
		t.Errorf("Longitude = %v, want nil", *created.Longitude)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.Latitude != nil || got.Longitude != nil {
		t.Errorf("Get() coordinates = (%v, %v), want (nil, nil)", got.Latitude, got.Longitude)
	}
}

func TestLocationRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	l := testLocation(c.ID, "Edge Location")
	l.ID = bogusID
	l.CreatedAt = bogusTime
	l.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, l)
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

func TestLocationRepositoryCreateFailsWhenCustomerDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testLocation(uuid.New(), "Orphan Location")) // customer does not exist

	assertConflict(t, err)
}

func TestLocationRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testLocation(c.ID, "Jane's Service Address"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.CustomerID != created.CustomerID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestLocationRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestLocationRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testLocation(c.ID, "Alpha Location"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testLocation(c.ID, "Beta Location"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	locations, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]location.Location, len(locations))
	for _, l := range locations {
		found[l.ID] = l
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created location")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created location")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture Customer, which is a different table), so the list is
	// exactly these two, letting us also check the ORDER BY name.
	if len(locations) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(locations), locations)
	}
	if locations[0].Name != "Alpha Location" || locations[1].Name != "Beta Location" {
		t.Errorf("List() order = [%q, %q], want [Alpha Location, Beta Location]", locations[0].Name, locations[1].Name)
	}
}

// TestLocationRepositoryListByCustomerID proves the WHERE customer_id =
// $1 filter, the one thing List() itself cannot exercise: a second
// customer's Location must never appear in the first customer's results.
func TestLocationRepositoryListByCustomerID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	otherCustomer := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testLocation(c.ID, "Alpha Location"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testLocation(c.ID, "Beta Location"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if _, err := repo.Create(ctx, testLocation(otherCustomer.ID, "Other Customer's Location")); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	locations, err := repo.ListByCustomerID(ctx, c.ID)
	if err != nil {
		t.Fatalf("ListByCustomerID() = %v", err)
	}

	if len(locations) != 2 {
		t.Fatalf("len(ListByCustomerID()) = %d, want 2; got %+v", len(locations), locations)
	}
	if locations[0].ID != first.ID || locations[1].ID != second.ID {
		t.Errorf("ListByCustomerID() = [%v, %v], want [%v, %v] (ordered by name)",
			locations[0].ID, locations[1].ID, first.ID, second.ID)
	}
}

func TestLocationRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	otherCustomer := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testLocation(c.ID, "Old Name"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	lat, lng := 40.0, -90.0
	updated, err := repo.Update(ctx, location.Location{
		ID:         created.ID,
		CustomerID: otherCustomer.ID,
		Name:       "New Name",
		Type:       location.LocationTypeBilling,
		Status:     location.LocationStatusInactive,
		Address1:   "456 Elm St",
		Latitude:   &lat,
		Longitude:  &lng,
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.CustomerID != otherCustomer.ID {
		t.Errorf("CustomerID = %v, want %v (CustomerID must be mutable via Update)", updated.CustomerID, otherCustomer.ID)
	}
	if updated.Type != location.LocationTypeBilling {
		t.Errorf("Type = %q, want %q", updated.Type, location.LocationTypeBilling)
	}
	if updated.Status != location.LocationStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, location.LocationStatusInactive)
	}
	if updated.Address1 != "456 Elm St" {
		t.Errorf("Address1 = %q, want %q", updated.Address1, "456 Elm St")
	}
	if updated.Latitude == nil || *updated.Latitude != lat {
		t.Errorf("Latitude = %v, want %v", updated.Latitude, lat)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestLocationRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	ghost := testLocation(c.ID, "Ghost")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestLocationRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testLocation(c.ID, "Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestLocationRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewLocationRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestLocationRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewLocationRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testLocation(c.ID, "First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testLocation(c.ID, "Second"))
	assertConflict(t, err)
}

// TestCustomerRepositoryDeleteBlockedByExistingLocation lives here, not in
// internal/customer/postgres, so that package's existing test files stay
// untouched — the same reasoning
// internal/inventory/postgres/building_test.go already documents for why
// its equivalent test lives with the child, not the parent. It exercises
// CustomerRepository.Delete against the new foreign key this migration
// adds.
func TestCustomerRepositoryDeleteBlockedByExistingLocation(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	locationRepo := postgres.NewLocationRepository(q, clock.New(), id.New())
	if _, err := locationRepo.Create(ctx, testLocation(c.ID, "Blocking Location")); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	customerRepo := customerpostgres.NewCustomerRepository(q, clock.New(), id.New())

	err := customerRepo.Delete(ctx, c.ID)

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
