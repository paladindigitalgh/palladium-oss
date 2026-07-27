//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestRepository mirrors internal/inventory/postgres/site_test.go's
// helper of the same name: open a transaction against the real test
// database, build the repository under test on it, and roll the
// transaction back on cleanup so tests never leave data behind or observe
// each other's writes.
func newTestRepository(t *testing.T, ids id.Generator) (*postgres.CustomerRepository, context.Context) {
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

	return postgres.NewCustomerRepository(tx, clock.New(), ids), ctx
}

func testCustomer(name string) customer.Customer {
	return customer.Customer{
		Name:         name,
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
	}
}

func TestCustomerRepositoryCreate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, customer.Customer{
		Name:         "Jane Doe",
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
		Description:  "Long-time customer",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.Name != "Jane Doe" {
		t.Errorf("Name = %q, want %q", created.Name, "Jane Doe")
	}
	if created.CustomerType != customer.CustomerTypeResidential {
		t.Errorf("CustomerType = %q, want %q", created.CustomerType, customer.CustomerTypeResidential)
	}
	if created.Status != customer.CustomerStatusActive {
		t.Errorf("Status = %q, want %q", created.Status, customer.CustomerStatusActive)
	}
	if created.Description != "Long-time customer" {
		t.Errorf("Description = %q, want %q", created.Description, "Long-time customer")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestCustomerRepositoryCreatePersistsEachDefinedCustomerTypeAndStatus(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	types := []customer.CustomerType{
		customer.CustomerTypeResidential,
		customer.CustomerTypeBusiness,
		customer.CustomerTypeGovernment,
		customer.CustomerTypeInternal,
	}
	statuses := []customer.CustomerStatus{
		customer.CustomerStatusActive,
		customer.CustomerStatusInactive,
		customer.CustomerStatusArchived,
	}

	for i, ct := range types {
		st := statuses[i%len(statuses)]
		c := testCustomer(uuid.NewString())
		c.CustomerType = ct
		c.Status = st

		created, err := repo.Create(ctx, c)
		if err != nil {
			t.Fatalf("Create() (type %q, status %q) = %v", ct, st, err)
		}
		if created.CustomerType != ct {
			t.Errorf("CustomerType = %q, want %q", created.CustomerType, ct)
		}
		if created.Status != st {
			t.Errorf("Status = %q, want %q", created.Status, st)
		}

		got, err := repo.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Get() = %v", err)
		}
		if got.CustomerType != ct || got.Status != st {
			t.Errorf("Get() = {CustomerType: %q, Status: %q}, want {%q, %q}", got.CustomerType, got.Status, ct, st)
		}
	}
}

func TestCustomerRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	c := testCustomer("Edge Customer")
	c.ID = bogusID
	c.CreatedAt = bogusTime
	c.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, c)
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

func TestCustomerRepositoryGet(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testCustomer("Jane Doe"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	// Compared field by field rather than with == or reflect.DeepEqual:
	// time.Time is documented as unsafe to compare that way (it embeds an
	// optional monotonic reading), so timestamps use Equal explicitly.
	if got.ID != created.ID || got.Name != created.Name || got.CustomerType != created.CustomerType || got.Status != created.Status {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestCustomerRepositoryGetNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestCustomerRepositoryList(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	first, err := repo.Create(ctx, testCustomer("Alpha Customer"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testCustomer("Beta Customer"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	customers, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]customer.Customer, len(customers))
	for _, c := range customers {
		found[c.ID] = c
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created customer")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created customer")
	}

	// Both were created within this same rolled-back transaction, so the
	// list is exactly these two, letting us also check the ORDER BY name.
	if len(customers) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(customers), customers)
	}
	if customers[0].Name != "Alpha Customer" || customers[1].Name != "Beta Customer" {
		t.Errorf("List() order = [%q, %q], want [Alpha Customer, Beta Customer]", customers[0].Name, customers[1].Name)
	}
}

func TestCustomerRepositoryUpdate(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, customer.Customer{
		Name:         "Old Name",
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
		Description:  "Old Description",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, customer.Customer{
		ID:           created.ID,
		Name:         "New Name",
		CustomerType: customer.CustomerTypeBusiness,
		Status:       customer.CustomerStatusInactive,
		Description:  "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.CustomerType != customer.CustomerTypeBusiness {
		t.Errorf("CustomerType = %q, want %q", updated.CustomerType, customer.CustomerTypeBusiness)
	}
	if updated.Status != customer.CustomerStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, customer.CustomerStatusInactive)
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

func TestCustomerRepositoryUpdateNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	_, err := repo.Update(ctx, customer.Customer{
		ID:           uuid.New(),
		Name:         "Ghost",
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
	})

	assertNotFound(t, err)
}

func TestCustomerRepositoryDelete(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	created, err := repo.Create(ctx, testCustomer("Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestCustomerRepositoryDeleteNotFound(t *testing.T) {
	repo, ctx := newTestRepository(t, id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestCustomerRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	fixedID := uuid.New()
	repo, ctx := newTestRepository(t, id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testCustomer("First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testCustomer("Second"))
	if err == nil {
		t.Fatal("second Create() = nil, want a conflict error for the duplicate primary key")
	}
	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindConflict, err)
	}
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

// testConfig points at the same database.Config the rest of the test
// suite uses (see internal/inventory/postgres/site_test.go): local
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
