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
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
	providerpostgres "github.com/paladindigitalgh/palladium-oss/internal/provider/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/service/postgres/service_test.go. A WorkflowInstance's one
// required foreign key (ServiceID) needs a full fixture Service, which
// itself needs a fixture Location and Product — see createTestService
// below.
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

// createTestService creates a real Service row (and every fixture it
// requires) through internal/service/postgres and its own dependencies —
// not internal/workflow/postgres — so a fixture failure surfaces as a
// clear failure of that domain's own Create, not a confusing failure
// somewhere else. This is the one place this package imports those
// domains at all: the domain model (internal/workflow) never does (see
// its package doc comment), only this test, which genuinely needs a real
// services row for the foreign key to reference.
func createTestService(t *testing.T, ctx context.Context, q database.Querier) service.Service {
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

	catalogRepo := catalogpostgres.NewCatalogRepository(q, clock.New(), id.New())
	cat, err := catalogRepo.Create(ctx, catalog.ProductCatalog{
		Name:   "Fixture Catalog " + uuid.NewString(),
		Status: catalog.CatalogStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create catalog: %v", err)
	}

	providerRepo := providerpostgres.NewProviderRepository(q, clock.New(), id.New())
	pr, err := providerRepo.Create(ctx, provider.Provider{
		Name:   "Fixture Provider " + uuid.NewString(),
		Status: provider.StatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create provider: %v", err)
	}

	productRepo := productpostgres.NewProductRepository(q, clock.New(), id.New())
	p, err := productRepo.Create(ctx, product.Product{
		CatalogID:   cat.ID,
		ProviderID:  pr.ID,
		Name:        "Fixture Product " + uuid.NewString(),
		Category:    product.ProductCategoryInternet,
		ServiceType: product.ServiceTypeResidential,
		Status:      product.ProductStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create product: %v", err)
	}

	serviceRepo := servicepostgres.NewServiceRepository(q, clock.New(), id.New())
	svc, err := serviceRepo.Create(ctx, service.Service{
		LocationID: l.ID,
		ProductID:  p.ID,
		Status:     service.ServiceStatusPending,
	})
	if err != nil {
		t.Fatalf("fixture: create service: %v", err)
	}
	return svc
}

func testInstance(serviceID uuid.UUID, definitionName string) workflow.Instance {
	return workflow.Instance{
		ServiceID:      serviceID,
		DefinitionName: definitionName,
		Status:         workflow.StatusPending,
	}
}

func TestRepositoryNextPendingReturnsOldestFirst(t *testing.T) {
	q, ctx := newTestQuerier(t)
	svc := createTestService(t, ctx, q)
	repo := postgres.NewRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testInstance(svc.ID, "provision-service"))
	if err != nil {
		t.Fatalf("Create() first = %v", err)
	}
	second, err := repo.Create(ctx, testInstance(svc.ID, "suspend-service"))
	if err != nil {
		t.Fatalf("Create() second = %v", err)
	}

	next, ok, err := repo.NextPending(ctx)
	if err != nil {
		t.Fatalf("NextPending() error = %v", err)
	}
	if !ok {
		t.Fatal("NextPending() ok = false, want true")
	}
	if next.ID != first.ID {
		t.Errorf("NextPending() = %v, want the oldest instance %v (not %v)", next.ID, first.ID, second.ID)
	}
}

func TestRepositoryNextPendingReturnsFalseWhenNoneArePending(t *testing.T) {
	q, ctx := newTestQuerier(t)
	svc := createTestService(t, ctx, q)
	repo := postgres.NewRepository(q, clock.New(), id.New())

	instance := testInstance(svc.ID, "provision-service")
	instance.Status = workflow.StatusSucceeded
	if _, err := repo.Create(ctx, instance); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	_, ok, err := repo.NextPending(ctx)
	if err != nil {
		t.Fatalf("NextPending() error = %v", err)
	}
	if ok {
		t.Error("NextPending() ok = true, want false when no instance is Pending")
	}
}

// testConfig points at the same database.Config the rest of the test
// suite uses (see internal/service/postgres/service_test.go): local
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
