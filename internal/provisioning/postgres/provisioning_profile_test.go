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
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
	providerpostgres "github.com/paladindigitalgh/palladium-oss/internal/provider/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/product/postgres/product_test.go. Every ProvisioningProfile
// test needs a fixture Product (which itself needs a fixture Catalog) to
// satisfy the required ProductID foreign key, and the fixtures must share
// the same transaction as the repository under test.
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

// createTestProduct creates a real Product row (via fixture Catalog and
// Provider rows) through internal/product/postgres,
// internal/catalog/postgres, and internal/provider/postgres — not this
// package — so a fixture failure surfaces as a clear failure of one of
// those domains' own Create, not a confusing failure somewhere else.
// This is the one place this package imports internal/product,
// internal/catalog, or internal/provider at all: the domain model
// (internal/provisioning) never does (see its package doc comment), only
// this test, which genuinely needs real products/providers rows for the
// foreign key to reference.
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
		CatalogID:  c.ID,
		ProviderID: pr.ID,
		Name:       "Fixture Product " + uuid.NewString(),
		Category:   product.ProductCategoryInternet,
		Status:     product.ProductStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create product: %v", err)
	}
	return p
}

func testProvisioningProfile(productID uuid.UUID, vendor, profileName string) provisioning.ProvisioningProfile {
	return provisioning.ProvisioningProfile{
		ProductID:   productID,
		Vendor:      vendor,
		ProfileName: profileName,
	}
}

func TestProvisioningProfileRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	profileName := "RES-500M-" + uuid.NewString()
	created, err := repo.Create(ctx, provisioning.ProvisioningProfile{
		ProductID:   p.ID,
		Vendor:      "Kontron",
		ProfileName: profileName,
		Description: "500 Mbps residential rate-limit + VLAN profile",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.ProductID != p.ID {
		t.Errorf("ProductID = %v, want %v", created.ProductID, p.ID)
	}
	if created.Vendor != "Kontron" {
		t.Errorf("Vendor = %q, want %q", created.Vendor, "Kontron")
	}
	if created.ProfileName != profileName {
		t.Errorf("ProfileName = %q, want %q", created.ProfileName, profileName)
	}
	if created.Description != "500 Mbps residential rate-limit + VLAN profile" {
		t.Errorf("Description = %q, want %q", created.Description, "500 Mbps residential rate-limit + VLAN profile")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestProvisioningProfileRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	profile := testProvisioningProfile(p.ID, "Kontron", "RES-100M-"+uuid.NewString())
	profile.ID = bogusID
	profile.CreatedAt = bogusTime
	profile.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, profile)
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

func TestProvisioningProfileRepositoryCreateFailsWhenProductDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testProvisioningProfile(uuid.New(), "Kontron", "RES-100M-"+uuid.NewString())) // product does not exist

	assertConflict(t, err)
}

func TestProvisioningProfileRepositoryCreateConflictOnDuplicateProductAndVendor(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	if _, err := repo.Create(ctx, testProvisioningProfile(p.ID, "Kontron", "RES-100M-"+uuid.NewString())); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	// Same Product, same vendor, a different profile name -- still a
	// conflict, because a Product may have at most one profile per
	// vendor (see the migration's UNIQUE (product_id, vendor)).
	_, err := repo.Create(ctx, testProvisioningProfile(p.ID, "Kontron", "RES-250M-"+uuid.NewString()))
	assertConflict(t, err)
}

func TestProvisioningProfileRepositoryCreateConflictOnDuplicateVendorAndProfileName(t *testing.T) {
	q, ctx := newTestQuerier(t)
	first := createTestProduct(t, ctx, q)
	second := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	profileName := "RES-500M-" + uuid.NewString()
	if _, err := repo.Create(ctx, testProvisioningProfile(first.ID, "Kontron", profileName)); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	// A different Product claiming the same vendor+profile name is a
	// conflict too (see the migration's UNIQUE (vendor, profile_name)):
	// two Products cannot both point at one OLT profile.
	_, err := repo.Create(ctx, testProvisioningProfile(second.ID, "Kontron", profileName))
	assertConflict(t, err)
}

func TestProvisioningProfileRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProvisioningProfile(p.ID, "Kontron", "RES-500M-"+uuid.NewString()))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.ProductID != created.ProductID || got.ProfileName != created.ProfileName {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestProvisioningProfileRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestProvisioningProfileRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testProvisioningProfile(p.ID, "Aardvark", "PROFILE-A"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testProvisioningProfile(p.ID, "Kontron", "PROFILE-B"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	profiles, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]provisioning.ProvisioningProfile, len(profiles))
	for _, pr := range profiles {
		found[pr.ID] = pr
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created profile")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created profile")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture Catalog/Product rows, different tables), so the list
	// is exactly these two, letting us also check the ORDER BY vendor.
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(profiles), profiles)
	}
	if profiles[0].Vendor != "Aardvark" || profiles[1].Vendor != "Kontron" {
		t.Errorf("List() order = [%q, %q], want [Aardvark, Kontron]", profiles[0].Vendor, profiles[1].Vendor)
	}
}

func TestProvisioningProfileRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	otherProduct := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProvisioningProfile(p.ID, "Kontron", "RES-100M-"+uuid.NewString()))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	newProfileName := "RES-250M-" + uuid.NewString()
	updated, err := repo.Update(ctx, provisioning.ProvisioningProfile{
		ID:          created.ID,
		ProductID:   otherProduct.ID,
		Vendor:      "Kontron",
		ProfileName: newProfileName,
		Description: "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.ProfileName != newProfileName {
		t.Errorf("ProfileName = %q, want %q", updated.ProfileName, newProfileName)
	}
	if updated.ProductID != otherProduct.ID {
		t.Errorf("ProductID = %v, want %v (ProductID must be mutable via Update)", updated.ProductID, otherProduct.ID)
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

func TestProvisioningProfileRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	ghost := testProvisioningProfile(p.ID, "Kontron", "GHOST")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestProvisioningProfileRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProvisioningProfile(p.ID, "Kontron", "TEMPORARY"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestProvisioningProfileRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

// TestProductRepositoryDeleteBlockedByExistingProvisioningProfile lives
// here, not in internal/product/postgres, so that package's existing
// test files stay untouched — the same reasoning
// internal/product/postgres/product_test.go already documents for why
// its equivalent test (blocking a Catalog delete) lives with the child,
// not the parent. It exercises ProductRepository.Delete against the
// foreign key this package's migration adds.
func TestProductRepositoryDeleteBlockedByExistingProvisioningProfile(t *testing.T) {
	q, ctx := newTestQuerier(t)
	p := createTestProduct(t, ctx, q)
	profileRepo := postgres.NewProvisioningProfileRepository(q, clock.New(), id.New())
	if _, err := profileRepo.Create(ctx, testProvisioningProfile(p.ID, "Kontron", "RES-500M-"+uuid.NewString())); err != nil {
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
