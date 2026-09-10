//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer"
	devicemanufacturerpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemodel"
	devicemodelpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemodel/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/serviceequipment/postgres/service_equipment_test.go.
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

// createTestCustomer creates a real Customer row, the simpler fixture
// this package needs compared to
// internal/serviceequipment/postgres/service_equipment_test.go's
// createTestService: a CustomerDevice references a Customer directly,
// with no Location/Product/Catalog chain in between.
func createTestCustomer(t *testing.T, ctx context.Context, q database.Querier) customer.Customer {
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
	return c
}

// createTestDevice creates a real Device row (and the fixture Device
// Manufacturer/Model chain it requires), mirroring
// internal/serviceequipment/postgres/service_equipment_test.go's own
// createTestDevice exactly.
func createTestDevice(t *testing.T, ctx context.Context, q database.Querier) inventory.Device {
	t.Helper()

	manufacturerRepo := devicemanufacturerpostgres.NewDeviceManufacturerRepository(q, clock.New(), id.New())
	manufacturer, err := manufacturerRepo.Create(ctx, devicemanufacturer.DeviceManufacturer{
		Name: "Fixture Manufacturer " + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("fixture: create device manufacturer: %v", err)
	}

	modelRepo := devicemodelpostgres.NewDeviceModelRepository(q, clock.New(), id.New())
	model, err := modelRepo.Create(ctx, devicemodel.DeviceModel{
		ManufacturerID: manufacturer.ID,
		Name:           "Fixture Model " + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("fixture: create device model: %v", err)
	}

	deviceRepo := inventorypostgres.NewDeviceRepository(q, clock.New(), id.New())
	d, err := deviceRepo.Create(ctx, inventory.Device{
		Metadata:      inventory.Metadata{Name: "Fixture Device " + uuid.NewString()},
		DeviceModelID: model.ID,
		SerialNumber:  uuid.NewString(),
		Status:        inventory.DeviceStatusUnused,
	})
	if err != nil {
		t.Fatalf("fixture: create device: %v", err)
	}
	return d
}

func testCustomerDevice(customerID, deviceID uuid.UUID) customerdevice.CustomerDevice {
	return customerdevice.CustomerDevice{
		CustomerID: customerID,
		DeviceID:   deviceID,
	}
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error is not an *apperror.Error: %v", err)
	}
	if appErr.Kind != apperror.KindNotFound {
		t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindNotFound)
	}
}

func assertConflict(t *testing.T, err error) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error is not an *apperror.Error: %v", err)
	}
	if appErr.Kind != apperror.KindConflict {
		t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindConflict)
	}
}

func TestCustomerDeviceRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	attachedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, customerdevice.CustomerDevice{
		CustomerID: c.ID,
		DeviceID:   d.ID,
		AttachedAt: &attachedAt,
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
	if created.DeviceID != d.ID {
		t.Errorf("DeviceID = %v, want %v", created.DeviceID, d.ID)
	}
	if created.AttachedAt == nil || !created.AttachedAt.Equal(attachedAt) {
		t.Errorf("AttachedAt = %v, want %v", created.AttachedAt, attachedAt)
	}
	if created.DetachedAt != nil {
		t.Errorf("DetachedAt = %v, want nil", created.DetachedAt)
	}
	if !created.Active() {
		t.Error("Active() = false for a newly created attachment, want true")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestCustomerDeviceRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	cd := testCustomerDevice(c.ID, d.ID)
	cd.ID = bogusID
	cd.CreatedAt = bogusTime
	cd.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, cd)
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

func TestCustomerDeviceRepositoryCreateFailsWhenCustomerDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testCustomerDevice(uuid.New(), d.ID)) // customer does not exist

	assertConflict(t, err)
}

func TestCustomerDeviceRepositoryCreateFailsWhenDeviceDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testCustomerDevice(c.ID, uuid.New())) // device does not exist

	assertConflict(t, err)
}

func TestCustomerDeviceRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testCustomerDevice(c.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.CustomerID != created.CustomerID || got.DeviceID != created.DeviceID {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
}

func TestCustomerDeviceRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestCustomerDeviceRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d1 := createTestDevice(t, ctx, q)
	d2 := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testCustomerDevice(c.ID, d1.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testCustomerDevice(c.ID, d2.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	records, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	var foundFirst, foundSecond bool
	for _, r := range records {
		if r.ID == first.ID {
			foundFirst = true
		}
		if r.ID == second.ID {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Errorf("List() = %+v, want both %v and %v present", records, first.ID, second.ID)
	}
}

func TestCustomerDeviceRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testCustomerDevice(c.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	detachedAt := time.Now()
	toUpdate := created
	toUpdate.DetachedAt = &detachedAt

	updated, err := repo.Update(ctx, toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Active() {
		t.Error("Active() = true after setting DetachedAt, want false")
	}
	if updated.CreatedAt.IsZero() {
		t.Error("Update() cleared CreatedAt")
	}
}

func TestCustomerDeviceRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	cd := testCustomerDevice(c.ID, d.ID)
	cd.ID = uuid.New()

	_, err := repo.Update(ctx, cd)

	assertNotFound(t, err)
}

func TestCustomerDeviceRepositoryGetActiveByDeviceID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testCustomerDevice(c.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	active, err := repo.GetActiveByDeviceID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetActiveByDeviceID() = %v", err)
	}
	if active.ID != created.ID {
		t.Errorf("GetActiveByDeviceID() ID = %v, want %v", active.ID, created.ID)
	}
}

func TestCustomerDeviceRepositoryGetActiveByDeviceIDNotFoundAfterDetach(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewCustomerDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testCustomerDevice(c.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	detachedAt := time.Now()
	toUpdate := created
	toUpdate.DetachedAt = &detachedAt
	if _, err := repo.Update(ctx, toUpdate); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	_, err = repo.GetActiveByDeviceID(ctx, d.ID)

	assertNotFound(t, err)
}
