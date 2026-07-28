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
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/postgres"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/serviceequipment/postgres/service_equipment_test.go.
// ProvisioningJob needs a fixture Service (itself needing a fixture
// Location -> Customer and Product -> Catalog) AND a fixture auth.User,
// all sharing this one transaction.
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

// createTestService creates a real Service row (and the fixture Location/
// Customer/Product/Catalog chain it requires) through
// internal/service/postgres and its own dependencies — not
// internal/provisioning/postgres — so a ProvisioningJob fixture failure
// surfaces as a clear failure of one specific layer, not a confusing
// failure somewhere else. This is the one place this package imports
// internal/service (and everything it depends on) at all: the domain
// model (internal/provisioning) never does (see its package doc
// comment), only this test, which genuinely needs a real services row
// for the foreign key to reference.
func createTestService(t *testing.T, ctx context.Context, q database.Querier) domainservice.Service {
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

	productRepo := productpostgres.NewProductRepository(q, clock.New(), id.New())
	p, err := productRepo.Create(ctx, product.Product{
		CatalogID: cat.ID,
		Name:      "Fixture Product " + uuid.NewString(),
		Category:  product.ProductCategoryInternet,
		Status:    product.ProductStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create product: %v", err)
	}

	serviceRepo := servicepostgres.NewServiceRepository(q, clock.New(), id.New())
	s, err := serviceRepo.Create(ctx, domainservice.Service{
		LocationID: l.ID,
		ProductID:  p.ID,
		Status:     domainservice.ServiceStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create service: %v", err)
	}
	return s
}

// createTestUser creates a real User row through internal/auth/postgres —
// see createTestService's doc comment for the same reasoning, applied to
// the other foreign key.
func createTestUser(t *testing.T, ctx context.Context, q database.Querier) auth.User {
	t.Helper()

	userRepo := authpostgres.NewUserRepository(q, clock.New(), id.New())
	u, err := userRepo.Create(ctx, auth.User{
		Email:        "fixture-" + uuid.NewString() + "@example.com",
		PasswordHash: "fixture-hash",
		Role:         auth.RoleOperator,
	})
	if err != nil {
		t.Fatalf("fixture: create user: %v", err)
	}
	return u
}

func testProvisioningJob(serviceID uuid.UUID) provisioning.ProvisioningJob {
	return provisioning.ProvisioningJob{
		ServiceID: serviceID,
		Operation: provisioning.ProvisioningOperationProvision,
		Status:    provisioning.ProvisioningStatusPending,
	}
}

func TestProvisioningRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	u := createTestUser(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	startedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, provisioning.ProvisioningJob{
		ServiceID:         s.ID,
		RequestedByUserID: &u.ID,
		Operation:         provisioning.ProvisioningOperationProvision,
		Status:            provisioning.ProvisioningStatusRunning,
		RetryCount:        2,
		StartedAt:         &startedAt,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.ServiceID != s.ID {
		t.Errorf("ServiceID = %v, want %v", created.ServiceID, s.ID)
	}
	if created.RequestedByUserID == nil || *created.RequestedByUserID != u.ID {
		t.Errorf("RequestedByUserID = %v, want %v", created.RequestedByUserID, u.ID)
	}
	if created.Operation != provisioning.ProvisioningOperationProvision {
		t.Errorf("Operation = %q, want %q", created.Operation, provisioning.ProvisioningOperationProvision)
	}
	if created.Status != provisioning.ProvisioningStatusRunning {
		t.Errorf("Status = %q, want %q", created.Status, provisioning.ProvisioningStatusRunning)
	}
	// RetryCount round-trip: goal 10 asks this be explicitly verified.
	if created.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want %d", created.RetryCount, 2)
	}
	if created.StartedAt == nil || !created.StartedAt.Equal(startedAt) {
		t.Errorf("StartedAt = %v, want %v", created.StartedAt, startedAt)
	}
	if created.ErrorMessage != nil {
		t.Errorf("ErrorMessage = %v, want nil", created.ErrorMessage)
	}
	if created.CompletedAt != nil {
		t.Errorf("CompletedAt = %v, want nil", created.CompletedAt)
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

// TestProvisioningRepositoryCreateWithoutOptionalFields proves
// RequestedByUserID, ErrorMessage, StartedAt, and CompletedAt round-trip
// as nil, not silently becoming a zero value — the exact distinction
// pointer fields exist to preserve (see provisioning.ProvisioningJob's
// doc comment).
func TestProvisioningRepositoryCreateWithoutOptionalFields(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProvisioningJob(s.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.RequestedByUserID != nil {
		t.Errorf("RequestedByUserID = %v, want nil", created.RequestedByUserID)
	}
	if created.ErrorMessage != nil {
		t.Errorf("ErrorMessage = %v, want nil", created.ErrorMessage)
	}
	if created.StartedAt != nil || created.CompletedAt != nil {
		t.Errorf("lifecycle timestamps = (%v, %v), want (nil, nil)", created.StartedAt, created.CompletedAt)
	}
	if created.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", created.RetryCount)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.RequestedByUserID != nil || got.ErrorMessage != nil || got.StartedAt != nil || got.CompletedAt != nil {
		t.Errorf("Get() optional fields = (%v, %v, %v, %v), want all nil",
			got.RequestedByUserID, got.ErrorMessage, got.StartedAt, got.CompletedAt)
	}
}

func TestProvisioningRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	j := testProvisioningJob(s.ID)
	j.ID = bogusID
	j.CreatedAt = bogusTime
	j.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, j)
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

func TestProvisioningRepositoryCreateFailsWhenServiceDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testProvisioningJob(uuid.New())) // service does not exist

	assertConflict(t, err)
}

func TestProvisioningRepositoryCreateFailsWhenRequestedByUserDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	ghostUserID := uuid.New()
	j := testProvisioningJob(s.ID)
	j.RequestedByUserID = &ghostUserID

	_, err := repo.Create(ctx, j)

	assertConflict(t, err)
}

func TestProvisioningRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProvisioningJob(s.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.ServiceID != created.ServiceID {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestProvisioningRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestProvisioningRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testProvisioningJob(s.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testProvisioningJob(s.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	jobs, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]provisioning.ProvisioningJob, len(jobs))
	for _, j := range jobs {
		found[j.ID] = j
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created job")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created job")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture chain, all different tables), so the list is exactly
	// these two, letting us also check the ORDER BY created_at.
	if len(jobs) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(jobs), jobs)
	}
	if jobs[0].ID != first.ID || jobs[1].ID != second.ID {
		t.Errorf("List() order = [%v, %v], want [%v, %v] (oldest first)", jobs[0].ID, jobs[1].ID, first.ID, second.ID)
	}
}

// TestProvisioningRepositoryListByServiceIDReturnsOnlyThatServicesJobs is
// goal 10's explicit requirement: ListByServiceID must return only the
// requested Service's jobs, not every job in the table.
func TestProvisioningRepositoryListByServiceIDReturnsOnlyThatServicesJobs(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s1 := createTestService(t, ctx, q)
	s2 := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	forS1, err := repo.Create(ctx, testProvisioningJob(s1.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if _, err := repo.Create(ctx, testProvisioningJob(s2.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	jobs, err := repo.ListByServiceID(ctx, s1.ID)
	if err != nil {
		t.Fatalf("ListByServiceID() = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("len(ListByServiceID(s1)) = %d, want 1; got %+v", len(jobs), jobs)
	}
	if jobs[0].ID != forS1.ID {
		t.Errorf("ListByServiceID(s1) = %+v, want the job created for s1 (%v)", jobs[0], forS1.ID)
	}
	if jobs[0].ServiceID != s1.ID {
		t.Errorf("returned job's ServiceID = %v, want %v", jobs[0].ServiceID, s1.ID)
	}
}

func TestProvisioningRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	u := createTestUser(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProvisioningJob(s.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	startedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 1, 15, 9, 5, 0, 0, time.UTC)
	errMsg := "device unreachable"
	updated, err := repo.Update(ctx, provisioning.ProvisioningJob{
		ID:                created.ID,
		ServiceID:         s.ID,
		RequestedByUserID: &u.ID,
		Operation:         provisioning.ProvisioningOperationProvision,
		Status:            provisioning.ProvisioningStatusFailed,
		RetryCount:        1,
		ErrorMessage:      &errMsg,
		StartedAt:         &startedAt,
		CompletedAt:       &completedAt,
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Status != provisioning.ProvisioningStatusFailed {
		t.Errorf("Status = %q, want %q", updated.Status, provisioning.ProvisioningStatusFailed)
	}
	if updated.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want %d", updated.RetryCount, 1)
	}
	if updated.ErrorMessage == nil || *updated.ErrorMessage != errMsg {
		t.Errorf("ErrorMessage = %v, want %q", updated.ErrorMessage, errMsg)
	}
	if updated.RequestedByUserID == nil || *updated.RequestedByUserID != u.ID {
		t.Errorf("RequestedByUserID = %v, want %v (must be mutable via Update)", updated.RequestedByUserID, u.ID)
	}
	if updated.CompletedAt == nil || !updated.CompletedAt.Equal(completedAt) {
		t.Errorf("CompletedAt = %v, want %v", updated.CompletedAt, completedAt)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestProvisioningRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	ghost := testProvisioningJob(s.ID)
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestProvisioningRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testProvisioningJob(s.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestProvisioningRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestProvisioningRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewProvisioningRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testProvisioningJob(s.ID)); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testProvisioningJob(s.ID))
	assertConflict(t, err)
}

// TestServiceRepositoryDeleteBlockedByExistingProvisioningJob lives here,
// not in internal/service/postgres, so that package's existing test
// files stay untouched — the same reasoning
// internal/serviceequipment/postgres/service_equipment_test.go already
// documents for why its equivalent tests live with the child, not the
// parent. It exercises ServiceRepository.Delete against the foreign key
// this migration adds.
func TestServiceRepositoryDeleteBlockedByExistingProvisioningJob(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	jobRepo := postgres.NewProvisioningRepository(q, clock.New(), id.New())
	if _, err := jobRepo.Create(ctx, testProvisioningJob(s.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	serviceRepo := servicepostgres.NewServiceRepository(q, clock.New(), id.New())

	err := serviceRepo.Delete(ctx, s.ID)

	assertConflict(t, err)
}

// TestUserRepositoryDeleteBlockedByExistingProvisioningJob would be
// TestServiceRepositoryDeleteBlockedByExistingProvisioningJob's
// counterpart for the other foreign key, but internal/auth's
// UserRepository has no Delete method (user accounts are not deletable in
// this codebase at all — see internal/auth/repository.go) — so there is
// no equivalent test to write here. The requested_by_user_id foreign key
// is still real and still enforced (see
// TestProvisioningRepositoryCreateFailsWhenRequestedByUserDoesNotExist
// above); it simply cannot be exercised from the delete direction because
// nothing in this codebase can delete a User.

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
