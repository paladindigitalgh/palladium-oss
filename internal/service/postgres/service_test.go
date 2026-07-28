//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	catalogpostgres "github.com/paladindigitalgh/palladium-oss/internal/catalog/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	locationpostgres "github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/location/postgres/location_test.go and
// internal/product/postgres/product_test.go. Service is the first entity
// with two required foreign keys, so its fixtures run one level deeper
// than either of those: a fixture Service needs a fixture Location (which
// itself needs a fixture Customer) AND a fixture Product (which itself
// needs a fixture Catalog), all sharing this one transaction.
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

// createTestLocation creates a real Location row (and the fixture
// Customer it requires) through internal/location/postgres and
// internal/customer/postgres — not internal/service/postgres — so a
// Service fixture failure surfaces as a clear failure of Location's or
// Customer's own Create, not a confusing failure somewhere else. This is
// the one place this package imports internal/location/internal/customer
// at all: the domain model (internal/service) never does (see its
// package doc comment), only this test, which genuinely needs real rows
// for the foreign key to reference.
func createTestLocation(t *testing.T, ctx context.Context, q database.Querier) location.Location {
	t.Helper()

	customerRepo := customerpostgres.NewCustomerRepository(q, clock.New(), id.New())
	c, err := customerRepo.Create(ctx, customer.Customer{
		Name:         "Fixture Customer " + uuid.NewString(),
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create customer: %v", err)
	}

	locationRepo := locationpostgres.NewLocationRepository(q, clock.New(), id.New())
	l, err := locationRepo.Create(ctx, location.Location{
		CustomerID: c.ID,
		Name:       "Fixture Location " + uuid.NewString(),
		Type:       location.LocationTypeService,
		Status:     location.LocationStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create location: %v", err)
	}
	return l
}

// createTestProduct creates a real Product row (and the fixture Catalog
// it requires) through internal/product/postgres and
// internal/catalog/postgres — see createTestLocation's doc comment for
// the same reasoning, applied to the other foreign key.
func createTestProduct(t *testing.T, ctx context.Context, q database.Querier) product.Product {
	t.Helper()

	catalogRepo := catalogpostgres.NewCatalogRepository(q, clock.New(), id.New())
	c, err := catalogRepo.Create(ctx, catalog.ProductCatalog{
		Name:   "Fixture Catalog " + uuid.NewString(),
		Status: catalog.CatalogStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create catalog: %v", err)
	}

	productRepo := productpostgres.NewProductRepository(q, clock.New(), id.New())
	p, err := productRepo.Create(ctx, product.Product{
		CatalogID: c.ID,
		Name:      "Fixture Product " + uuid.NewString(),
		Category:  product.ProductCategoryInternet,
		Status:    product.ProductStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create product: %v", err)
	}
	return p
}

func testService(locationID, productID uuid.UUID) service.Service {
	return service.Service{
		LocationID: locationID,
		ProductID:  productID,
		Status:     service.ServiceStatusPending,
	}
}

func TestServiceRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	activatedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, service.Service{
		LocationID:  l.ID,
		ProductID:   p.ID,
		Status:      service.ServiceStatusActive,
		Description: "Primary residential internet service",
		ActivatedAt: &activatedAt,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.LocationID != l.ID {
		t.Errorf("LocationID = %v, want %v", created.LocationID, l.ID)
	}
	if created.ProductID != p.ID {
		t.Errorf("ProductID = %v, want %v", created.ProductID, p.ID)
	}
	if created.Status != service.ServiceStatusActive {
		t.Errorf("Status = %q, want %q", created.Status, service.ServiceStatusActive)
	}
	if created.Description != "Primary residential internet service" {
		t.Errorf("Description = %q, want %q", created.Description, "Primary residential internet service")
	}
	if created.ActivatedAt == nil || !created.ActivatedAt.Equal(activatedAt) {
		t.Errorf("ActivatedAt = %v, want %v", created.ActivatedAt, activatedAt)
	}
	if created.SuspendedAt != nil {
		t.Errorf("SuspendedAt = %v, want nil", created.SuspendedAt)
	}
	if created.DisconnectedAt != nil {
		t.Errorf("DisconnectedAt = %v, want nil", created.DisconnectedAt)
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

// TestServiceRepositoryCreateWithoutLifecycleTimestamps proves
// ActivatedAt/SuspendedAt/DisconnectedAt round-trip as nil, not silently
// becoming the zero time — the exact distinction *time.Time exists to
// preserve (see service.Service's doc comment).
func TestServiceRepositoryCreateWithoutLifecycleTimestamps(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testService(l.ID, p.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ActivatedAt != nil || created.SuspendedAt != nil || created.DisconnectedAt != nil {
		t.Errorf("lifecycle timestamps = (%v, %v, %v), want (nil, nil, nil)",
			created.ActivatedAt, created.SuspendedAt, created.DisconnectedAt)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ActivatedAt != nil || got.SuspendedAt != nil || got.DisconnectedAt != nil {
		t.Errorf("Get() lifecycle timestamps = (%v, %v, %v), want (nil, nil, nil)",
			got.ActivatedAt, got.SuspendedAt, got.DisconnectedAt)
	}
}

func TestServiceRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	s := testService(l.ID, p.ID)
	s.ID = bogusID
	s.CreatedAt = bogusTime
	s.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, s)
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

func TestServiceRepositoryCreateFailsWhenLocationDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testService(uuid.New(), p.ID)) // location does not exist

	assertConflict(t, err)
}

func TestServiceRepositoryCreateFailsWhenProductDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testService(l.ID, uuid.New())) // product does not exist

	assertConflict(t, err)
}

func TestServiceRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testService(l.ID, p.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.LocationID != created.LocationID || got.ProductID != created.ProductID {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestServiceRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestServiceRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testService(l.ID, p.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testService(l.ID, p.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	services, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]service.Service, len(services))
	for _, s := range services {
		found[s.ID] = s
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created service")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created service")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture Location/Customer/Product/Catalog, all different
	// tables), so the list is exactly these two, letting us also check
	// the ORDER BY created_at.
	if len(services) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(services), services)
	}
	if services[0].ID != first.ID || services[1].ID != second.ID {
		t.Errorf("List() order = [%v, %v], want [%v, %v] (oldest first)",
			services[0].ID, services[1].ID, first.ID, second.ID)
	}
}

func TestServiceRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	otherLocation := createTestLocation(t, ctx, q)
	otherProduct := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testService(l.ID, p.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	activatedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	suspendedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	updated, err := repo.Update(ctx, service.Service{
		ID:          created.ID,
		LocationID:  otherLocation.ID,
		ProductID:   otherProduct.ID,
		Status:      service.ServiceStatusSuspended,
		Description: "Suspended for non-payment",
		ActivatedAt: &activatedAt,
		SuspendedAt: &suspendedAt,
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.LocationID != otherLocation.ID {
		t.Errorf("LocationID = %v, want %v (LocationID must be mutable via Update)", updated.LocationID, otherLocation.ID)
	}
	if updated.ProductID != otherProduct.ID {
		t.Errorf("ProductID = %v, want %v (ProductID must be mutable via Update)", updated.ProductID, otherProduct.ID)
	}
	if updated.Status != service.ServiceStatusSuspended {
		t.Errorf("Status = %q, want %q", updated.Status, service.ServiceStatusSuspended)
	}
	if updated.Description != "Suspended for non-payment" {
		t.Errorf("Description = %q, want %q", updated.Description, "Suspended for non-payment")
	}
	if updated.SuspendedAt == nil || !updated.SuspendedAt.Equal(suspendedAt) {
		t.Errorf("SuspendedAt = %v, want %v", updated.SuspendedAt, suspendedAt)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestServiceRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	ghost := testService(l.ID, p.ID)
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestServiceRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testService(l.ID, p.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestServiceRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewServiceRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestServiceRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewServiceRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testService(l.ID, p.ID)); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testService(l.ID, p.ID))
	assertConflict(t, err)
}

// TestLocationRepositoryDeleteBlockedByExistingService lives here, not in
// internal/location/postgres, so that package's existing test files stay
// untouched — the same reasoning
// internal/location/postgres/location_test.go already documents for why
// its equivalent test (blocking a Customer delete) lives with the child,
// not the parent. It exercises LocationRepository.Delete against the
// foreign key this migration adds.
func TestLocationRepositoryDeleteBlockedByExistingService(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	serviceRepo := postgres.NewServiceRepository(q, clock.New(), id.New())
	if _, err := serviceRepo.Create(ctx, testService(l.ID, p.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	locationRepo := locationpostgres.NewLocationRepository(q, clock.New(), id.New())

	err := locationRepo.Delete(ctx, l.ID)

	assertConflict(t, err)
}

// TestProductRepositoryDeleteBlockedByExistingService is
// TestLocationRepositoryDeleteBlockedByExistingService's counterpart for
// the other foreign key.
func TestProductRepositoryDeleteBlockedByExistingService(t *testing.T) {
	q, ctx := newTestQuerier(t)
	l := createTestLocation(t, ctx, q)
	p := createTestProduct(t, ctx, q)
	serviceRepo := postgres.NewServiceRepository(q, clock.New(), id.New())
	if _, err := serviceRepo.Create(ctx, testService(l.ID, p.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	productRepo := productpostgres.NewProductRepository(q, clock.New(), id.New())

	err := productRepo.Delete(ctx, p.ID)

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
