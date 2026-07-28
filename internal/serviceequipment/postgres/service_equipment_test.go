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
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	locationpostgres "github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/service/postgres/service_test.go. ServiceEquipment needs the
// deepest fixture chain in this codebase so far: a fixture Service
// (itself needing a fixture Location -> Customer and Product -> Catalog)
// AND a fixture Device (via internal/inventory/postgres), all sharing
// this one transaction.
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
// internal/serviceequipment/postgres — so a ServiceEquipment fixture
// failure surfaces as a clear failure of one specific layer, not a
// confusing failure somewhere else. This is the one place this package
// imports internal/service (and everything it depends on) at all: the
// domain model (internal/serviceequipment) never does (see its package
// doc comment), only this test, which genuinely needs a real services row
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

// createTestDevice creates a real Device row through
// internal/inventory/postgres — see createTestService's doc comment for
// the same reasoning, applied to the other foreign key. RackID is left
// nil: inventory.Device's RackID is nullable (a device can exist before
// it is ever racked — see database/migrations/00006_inventory_devices.sql),
// and nothing about a ServiceEquipment assignment requires a Device to be
// racked.
func createTestDevice(t *testing.T, ctx context.Context, q database.Querier) inventory.Device {
	t.Helper()

	deviceRepo := inventorypostgres.NewDeviceRepository(q, clock.New(), id.New())
	d, err := deviceRepo.Create(ctx, inventory.Device{
		Metadata:     inventory.Metadata{Name: "Fixture Device " + uuid.NewString()},
		Manufacturer: "Fixture Manufacturer",
		Model:        "Fixture Model",
		SerialNumber: uuid.NewString(),
		Status:       inventory.DeviceStatusInstalled,
	})
	if err != nil {
		t.Fatalf("fixture: create device: %v", err)
	}
	return d
}

func testServiceEquipment(serviceID, deviceID uuid.UUID) serviceequipment.ServiceEquipment {
	return serviceequipment.ServiceEquipment{
		ServiceID: serviceID,
		DeviceID:  deviceID,
		Role:      serviceequipment.EquipmentRoleONU,
	}
}

func TestServiceEquipmentRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	installedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, serviceequipment.ServiceEquipment{
		ServiceID:   s.ID,
		DeviceID:    d.ID,
		Role:        serviceequipment.EquipmentRoleONU,
		Description: "Installed in the network closet",
		InstalledAt: &installedAt,
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
	if created.DeviceID != d.ID {
		t.Errorf("DeviceID = %v, want %v", created.DeviceID, d.ID)
	}
	if created.Role != serviceequipment.EquipmentRoleONU {
		t.Errorf("Role = %q, want %q", created.Role, serviceequipment.EquipmentRoleONU)
	}
	if created.Description != "Installed in the network closet" {
		t.Errorf("Description = %q, want %q", created.Description, "Installed in the network closet")
	}
	if created.InstalledAt == nil || !created.InstalledAt.Equal(installedAt) {
		t.Errorf("InstalledAt = %v, want %v", created.InstalledAt, installedAt)
	}
	if created.RemovedAt != nil {
		t.Errorf("RemovedAt = %v, want nil", created.RemovedAt)
	}
	if !created.Active() {
		t.Error("Active() = false for a newly created assignment, want true")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestServiceEquipmentRepositoryCreateWithoutLifecycleTimestamps(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testServiceEquipment(s.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.InstalledAt != nil || created.RemovedAt != nil {
		t.Errorf("lifecycle timestamps = (%v, %v), want (nil, nil)", created.InstalledAt, created.RemovedAt)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.InstalledAt != nil || got.RemovedAt != nil {
		t.Errorf("Get() lifecycle timestamps = (%v, %v), want (nil, nil)", got.InstalledAt, got.RemovedAt)
	}
}

func TestServiceEquipmentRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	e := testServiceEquipment(s.ID, d.ID)
	e.ID = bogusID
	e.CreatedAt = bogusTime
	e.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, e)
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

func TestServiceEquipmentRepositoryCreateFailsWhenServiceDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testServiceEquipment(uuid.New(), d.ID)) // service does not exist

	assertConflict(t, err)
}

func TestServiceEquipmentRepositoryCreateFailsWhenDeviceDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testServiceEquipment(s.ID, uuid.New())) // device does not exist

	assertConflict(t, err)
}

func TestServiceEquipmentRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testServiceEquipment(s.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.ServiceID != created.ServiceID || got.DeviceID != created.DeviceID {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestServiceEquipmentRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestServiceEquipmentRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d1 := createTestDevice(t, ctx, q)
	d2 := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testServiceEquipment(s.ID, d1.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testServiceEquipment(s.ID, d2.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	equipment, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]serviceequipment.ServiceEquipment, len(equipment))
	for _, e := range equipment {
		found[e.ID] = e
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created assignment")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created assignment")
	}

	// Both were created within this same rolled-back transaction (plus
	// the fixture chain, all different tables), so the list is exactly
	// these two, letting us also check the ORDER BY created_at.
	if len(equipment) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(equipment), equipment)
	}
	if equipment[0].ID != first.ID || equipment[1].ID != second.ID {
		t.Errorf("List() order = [%v, %v], want [%v, %v] (oldest first)",
			equipment[0].ID, equipment[1].ID, first.ID, second.ID)
	}
}

func TestServiceEquipmentRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	otherDevice := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testServiceEquipment(s.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	installedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	updated, err := repo.Update(ctx, serviceequipment.ServiceEquipment{
		ID:          created.ID,
		ServiceID:   s.ID,
		DeviceID:    otherDevice.ID,
		Role:        serviceequipment.EquipmentRoleRouter,
		Description: "Swapped for a replacement unit",
		InstalledAt: &installedAt,
		RemovedAt:   &removedAt,
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.DeviceID != otherDevice.ID {
		t.Errorf("DeviceID = %v, want %v (DeviceID must be mutable via Update)", updated.DeviceID, otherDevice.ID)
	}
	if updated.Role != serviceequipment.EquipmentRoleRouter {
		t.Errorf("Role = %q, want %q", updated.Role, serviceequipment.EquipmentRoleRouter)
	}
	if updated.Description != "Swapped for a replacement unit" {
		t.Errorf("Description = %q, want %q", updated.Description, "Swapped for a replacement unit")
	}
	if updated.RemovedAt == nil || !updated.RemovedAt.Equal(removedAt) {
		t.Errorf("RemovedAt = %v, want %v", updated.RemovedAt, removedAt)
	}
	if updated.Active() {
		t.Error("Active() = true after RemovedAt was set, want false")
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestServiceEquipmentRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	ghost := testServiceEquipment(s.ID, d.ID)
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestServiceEquipmentRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testServiceEquipment(s.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestServiceEquipmentRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestServiceEquipmentRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testServiceEquipment(s.ID, d.ID)); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	otherDevice := createTestDevice(t, ctx, q)
	_, err := repo.Create(ctx, testServiceEquipment(s.ID, otherDevice.ID))
	assertConflict(t, err)
}

// TestServiceEquipmentRepositoryGetActiveByDeviceIDReturnsActiveAssignment
// proves the query goal 4 asks for by name finds the one active (RemovedAt
// == nil) assignment for a Device.
func TestServiceEquipmentRepositoryGetActiveByDeviceIDReturnsActiveAssignment(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testServiceEquipment(s.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	active, err := repo.GetActiveByDeviceID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetActiveByDeviceID() = %v", err)
	}
	if active.ID != created.ID {
		t.Errorf("GetActiveByDeviceID() = %+v, want %+v", active, created)
	}
}

// TestServiceEquipmentRepositoryGetActiveByDeviceIDNotFoundWhenNoneActive
// proves GetActiveByDeviceID reports not-found both when a Device has
// never had an assignment, and when its only assignment has been removed
// — "historical assignments remain allowed" (goal 2) means a removed
// assignment must not still count as active.
func TestServiceEquipmentRepositoryGetActiveByDeviceIDNotFoundWhenNoneActive(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	_, err := repo.GetActiveByDeviceID(ctx, d.ID)
	assertNotFound(t, err)

	created, err := repo.Create(ctx, testServiceEquipment(s.ID, d.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	created.RemovedAt = &removedAt
	if _, err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	_, err = repo.GetActiveByDeviceID(ctx, d.ID)
	assertNotFound(t, err)
}

// TestServiceEquipmentRepositoryListActiveByServiceIDReturnsOnlyActiveAssignmentsForThatService
// proves ListActiveByServiceID (added for internal/provisioning/engine —
// see its doc comment on ServiceEquipmentRepository in
// internal/serviceequipment/repository.go) returns every active
// assignment for the requested Service, excludes a removed assignment
// for that same Service, and excludes active assignments belonging to a
// different Service entirely.
func TestServiceEquipmentRepositoryListActiveByServiceIDReturnsOnlyActiveAssignmentsForThatService(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	otherService := createTestService(t, ctx, q)
	d1 := createTestDevice(t, ctx, q)
	d2 := createTestDevice(t, ctx, q)
	d3 := createTestDevice(t, ctx, q)
	d4 := createTestDevice(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	activeForS, err := repo.Create(ctx, testServiceEquipment(s.ID, d1.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	secondActiveForS, err := repo.Create(ctx, testServiceEquipment(s.ID, d2.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	removedForS, err := repo.Create(ctx, testServiceEquipment(s.ID, d3.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	removedForS.RemovedAt = &removedAt
	if _, err := repo.Update(ctx, removedForS); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if _, err := repo.Create(ctx, testServiceEquipment(otherService.ID, d4.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	equipment, err := repo.ListActiveByServiceID(ctx, s.ID)
	if err != nil {
		t.Fatalf("ListActiveByServiceID() = %v", err)
	}

	found := make(map[uuid.UUID]bool, len(equipment))
	for _, e := range equipment {
		found[e.ID] = true
		if !e.Active() {
			t.Errorf("ListActiveByServiceID() returned an inactive record: %+v", e)
		}
		if e.ServiceID != s.ID {
			t.Errorf("ListActiveByServiceID(%v) returned equipment for a different service: %+v", s.ID, e)
		}
	}
	if len(equipment) != 2 {
		t.Fatalf("len(ListActiveByServiceID()) = %d, want 2; got %+v", len(equipment), equipment)
	}
	if !found[activeForS.ID] || !found[secondActiveForS.ID] {
		t.Errorf("ListActiveByServiceID() = %+v, want both %v and %v", equipment, activeForS.ID, secondActiveForS.ID)
	}
}

func TestServiceEquipmentRepositoryListActiveByServiceIDReturnsEmptyWhenNoneActive(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	repo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	equipment, err := repo.ListActiveByServiceID(ctx, s.ID)
	if err != nil {
		t.Fatalf("ListActiveByServiceID() = %v", err)
	}
	if len(equipment) != 0 {
		t.Errorf("len(ListActiveByServiceID()) = %d, want 0; got %+v", len(equipment), equipment)
	}
}

// TestServiceRepositoryDeleteBlockedByExistingServiceEquipment lives
// here, not in internal/service/postgres, so that package's existing test
// files stay untouched — the same reasoning
// internal/service/postgres/service_test.go already documents for why its
// equivalent tests (blocking Location/Product deletes) live with the
// child, not the parent. It exercises ServiceRepository.Delete against
// the foreign key this migration adds.
func TestServiceRepositoryDeleteBlockedByExistingServiceEquipment(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	equipmentRepo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())
	if _, err := equipmentRepo.Create(ctx, testServiceEquipment(s.ID, d.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	serviceRepo := servicepostgres.NewServiceRepository(q, clock.New(), id.New())

	err := serviceRepo.Delete(ctx, s.ID)

	assertConflict(t, err)
}

// TestDeviceRepositoryDeleteBlockedByExistingServiceEquipment is
// TestServiceRepositoryDeleteBlockedByExistingServiceEquipment's
// counterpart for the other foreign key.
func TestDeviceRepositoryDeleteBlockedByExistingServiceEquipment(t *testing.T) {
	q, ctx := newTestQuerier(t)
	s := createTestService(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	equipmentRepo := postgres.NewServiceEquipmentRepository(q, clock.New(), id.New())
	if _, err := equipmentRepo.Create(ctx, testServiceEquipment(s.ID, d.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	deviceRepo := inventorypostgres.NewDeviceRepository(q, clock.New(), id.New())

	err := deviceRepo.Delete(ctx, d.ID)

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
