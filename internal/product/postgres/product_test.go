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
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	"github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/location/postgres/location_test.go. Every Product test needs a
// fixture ProductCatalog to satisfy the required CatalogID foreign key,
// and the fixture must share the same transaction as the repository under
// test, so tests here call this directly rather than hiding it behind a
// Catalog-style newTestRepository wrapper.
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

// createTestCatalog creates a real ProductCatalog row through
// internal/catalog/postgres — not internal/product/postgres — so a
// Product fixture failure surfaces as a clear failure of Catalog's own
// Create, not a confusing failure somewhere else. This is the one place
// this package imports internal/catalog at all: the domain model
// (internal/product) never does (see its package doc comment), only this
// test, which genuinely needs a real catalogs row for the foreign key to
// reference.
func createTestCatalog(t *testing.T, ctx context.Context, q database.Querier) catalog.ProductCatalog {
	t.Helper()

	repo := catalogpostgres.NewCatalogRepository(q, clock.New(), id.New())
	c, err := repo.Create(ctx, catalog.ProductCatalog{
		Name:   "Fixture Catalog " + uuid.NewString(),
		Status: catalog.CatalogStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create catalog: %v", err)
	}
	return c
}

func testProduct(catalogID uuid.UUID, name string) product.Product {
	return product.Product{
		CatalogID: catalogID,
		Name:      name,
		Category:  product.ProductCategoryInternet,
		Status:    product.ProductStatusActive,
	}
}

func TestProductRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, product.Product{
		CatalogID:   c.ID,
		Name:        "Residential Internet 100/20",
		Category:    product.ProductCategoryInternet,
		Status:      product.ProductStatusActive,
		Description: "100 Mbps down / 20 Mbps up",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.CatalogID != c.ID {
		t.Errorf("CatalogID = %v, want %v", created.CatalogID, c.ID)
	}
	if created.Name != "Residential Internet 100/20" {
		t.Errorf("Name = %q, want %q", created.Name, "Residential Internet 100/20")
	}
	if created.Category != product.ProductCategoryInternet {
		t.Errorf("Category = %q, want %q", created.Category, product.ProductCategoryInternet)
	}
	if created.Status != product.ProductStatusActive {
		t.Errorf("Status = %q, want %q", created.Status, product.ProductStatusActive)
	}
	if created.Description != "100 Mbps down / 20 Mbps up" {
		t.Errorf("Description = %q, want %q", created.Description, "100 Mbps down / 20 Mbps up")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestProductRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	p := testProduct(c.ID, "Edge Product")
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

func TestProductRepositoryCreateFailsWhenCatalogDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testProduct(uuid.New(), "Orphan Product")) // catalog does not exist

	assertConflict(t, err)
}

func TestProductRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProduct(c.ID, "Residential Internet"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.CatalogID != created.CatalogID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestProductRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestProductRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testProduct(c.ID, "Alpha Product"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testProduct(c.ID, "Beta Product"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	products, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]product.Product, len(products))
	for _, p := range products {
		found[p.ID] = p
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created product")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created product")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture ProductCatalog, which is a different table), so the
	// list is exactly these two, letting us also check the ORDER BY name.
	if len(products) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(products), products)
	}
	if products[0].Name != "Alpha Product" || products[1].Name != "Beta Product" {
		t.Errorf("List() order = [%q, %q], want [Alpha Product, Beta Product]", products[0].Name, products[1].Name)
	}
}

func TestProductRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	otherCatalog := createTestCatalog(t, ctx, q)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProduct(c.ID, "Old Name"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, product.Product{
		ID:          created.ID,
		CatalogID:   otherCatalog.ID,
		Name:        "New Name",
		Category:    product.ProductCategoryVoice,
		Status:      product.ProductStatusRetired,
		Description: "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.CatalogID != otherCatalog.ID {
		t.Errorf("CatalogID = %v, want %v (CatalogID must be mutable via Update)", updated.CatalogID, otherCatalog.ID)
	}
	if updated.Category != product.ProductCategoryVoice {
		t.Errorf("Category = %q, want %q", updated.Category, product.ProductCategoryVoice)
	}
	if updated.Status != product.ProductStatusRetired {
		t.Errorf("Status = %q, want %q", updated.Status, product.ProductStatusRetired)
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

func TestProductRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	ghost := testProduct(c.ID, "Ghost")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestProductRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProduct(c.ID, "Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestProductRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProductRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestProductRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewProductRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testProduct(c.ID, "First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testProduct(c.ID, "Second"))
	assertConflict(t, err)
}

// TestCatalogRepositoryDeleteBlockedByExistingProduct lives here, not in
// internal/catalog/postgres, so that package's existing test files stay
// untouched — the same reasoning
// internal/location/postgres/location_test.go already documents for why
// its equivalent test (blocking a Customer delete) lives with the child,
// not the parent. It exercises CatalogRepository.Delete against the
// foreign key this migration adds.
func TestCatalogRepositoryDeleteBlockedByExistingProduct(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCatalog(t, ctx, q)
	productRepo := postgres.NewProductRepository(q, clock.New(), id.New())
	if _, err := productRepo.Create(ctx, testProduct(c.ID, "Blocking Product")); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	catalogRepo := catalogpostgres.NewCatalogRepository(q, clock.New(), id.New())

	err := catalogRepo.Delete(ctx, c.ID)

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
