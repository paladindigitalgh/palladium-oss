//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
	"github.com/paladindigitalgh/palladium-oss/internal/contact/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as every other
// repository test file in this codebase (see e.g.
// internal/location/postgres/location_test.go). Every Contact test needs
// a fixture Customer to satisfy the required CustomerID foreign key, and
// the fixture must share the same transaction as the repository under
// test, so tests here call this directly rather than hiding it behind a
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

// createTestCustomer creates a real Customer row through
// internal/customer/postgres — not internal/contact/postgres — so a
// Contact fixture failure surfaces as a clear failure of Customer's own
// Create, not a confusing failure somewhere else. This is the one place
// this package imports internal/customer at all: the domain model
// (internal/contact) never does (see its package doc comment), only this
// test, which genuinely needs a real customers row for the foreign key
// to reference.
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

func testContact(customerID uuid.UUID, name string) contact.Contact {
	return contact.Contact{
		CustomerID: customerID,
		Name:       name,
		Role:       contact.ContactRolePrimary,
		Status:     contact.ContactStatusActive,
	}
}

func TestContactRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, contact.Contact{
		CustomerID:  c.ID,
		Name:        "Jane Doe",
		Role:        contact.ContactRolePrimary,
		Email:       "jane@example.com",
		Phone:       "555-0100",
		Status:      contact.ContactStatusActive,
		Description: "Prefers email over phone",
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
	if created.Name != "Jane Doe" {
		t.Errorf("Name = %q, want %q", created.Name, "Jane Doe")
	}
	if created.Role != contact.ContactRolePrimary {
		t.Errorf("Role = %q, want %q", created.Role, contact.ContactRolePrimary)
	}
	if created.Status != contact.ContactStatusActive {
		t.Errorf("Status = %q, want %q", created.Status, contact.ContactStatusActive)
	}
	if created.Email != "jane@example.com" || created.Phone != "555-0100" {
		t.Errorf("email/phone = %+v, want Email=jane@example.com Phone=555-0100", created)
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestContactRepositoryCreateWithoutEmailOrPhone(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testContact(c.ID, "No Email Or Phone"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.Email != "" {
		t.Errorf("Email = %q, want empty", created.Email)
	}
	if created.Phone != "" {
		t.Errorf("Phone = %q, want empty", created.Phone)
	}
}

func TestContactRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	ct := testContact(c.ID, "Edge Contact")
	ct.ID = bogusID
	ct.CreatedAt = bogusTime
	ct.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, ct)
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

func TestContactRepositoryCreateFailsWhenCustomerDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testContact(uuid.New(), "Orphan Contact")) // customer does not exist

	assertConflict(t, err)
}

func TestContactRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testContact(c.ID, "Jane's Contact"))
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

func TestContactRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestContactRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testContact(c.ID, "Alpha Contact"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testContact(c.ID, "Beta Contact"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	contacts, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]contact.Contact, len(contacts))
	for _, c := range contacts {
		found[c.ID] = c
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created contact")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created contact")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture Customer, which is a different table), so the list is
	// exactly these two, letting us also check the ORDER BY name.
	if len(contacts) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(contacts), contacts)
	}
	if contacts[0].Name != "Alpha Contact" || contacts[1].Name != "Beta Contact" {
		t.Errorf("List() order = [%q, %q], want [Alpha Contact, Beta Contact]", contacts[0].Name, contacts[1].Name)
	}
}

func TestContactRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	otherCustomer := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testContact(c.ID, "Old Name"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, contact.Contact{
		ID:          created.ID,
		CustomerID:  otherCustomer.ID,
		Name:        "New Name",
		Role:        contact.ContactRoleBilling,
		Email:       "new@example.com",
		Phone:       "555-0199",
		Status:      contact.ContactStatusInactive,
		Description: "Updated",
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
	if updated.Role != contact.ContactRoleBilling {
		t.Errorf("Role = %q, want %q", updated.Role, contact.ContactRoleBilling)
	}
	if updated.Status != contact.ContactStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, contact.ContactStatusInactive)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("Email = %q, want %q", updated.Email, "new@example.com")
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestContactRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	ghost := testContact(c.ID, "Ghost")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestContactRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testContact(c.ID, "Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestContactRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewContactRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestContactRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewContactRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testContact(c.ID, "First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testContact(c.ID, "Second"))
	assertConflict(t, err)
}

// TestCustomerRepositoryDeleteCascadesToContacts lives here, not in
// internal/customer/postgres, so that package's existing test files stay
// untouched — the same reasoning
// internal/location/postgres/location_test.go's
// TestCustomerRepositoryDeleteBlockedByExistingLocation already documents
// for why its equivalent test lives with the child, not the parent. This
// is the one piece of real, non-mechanical behavior this domain adds:
// unlike Location (ON DELETE RESTRICT, proven to block a Customer delete
// in location_test.go), contacts.customer_id is ON DELETE CASCADE — a
// Customer with Contacts on file must delete successfully, and its
// Contacts must be gone afterward, not orphaned or blocking anything.
func TestCustomerRepositoryDeleteCascadesToContacts(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	contactRepo := postgres.NewContactRepository(q, clock.New(), id.New())
	created, err := contactRepo.Create(ctx, testContact(c.ID, "Should Be Cascaded"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	customerRepo := customerpostgres.NewCustomerRepository(q, clock.New(), id.New())

	if err := customerRepo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("Delete() = %v, want nil (Contacts must not block a Customer delete)", err)
	}

	_, err = contactRepo.Get(ctx, created.ID)
	assertNotFound(t, err)
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
// suite uses (see internal/location/postgres/location_test.go): local
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
