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
	"github.com/paladindigitalgh/palladium-oss/internal/contact"
	contactpostgres "github.com/paladindigitalgh/palladium-oss/internal/contact/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	customerdevicepostgres "github.com/paladindigitalgh/palladium-oss/internal/customerdevice/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer"
	devicemanufacturerpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemodel"
	devicemodelpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemodel/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	locationpostgres "github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
	providerpostgres "github.com/paladindigitalgh/palladium-oss/internal/provider/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/report"
	"github.com/paladindigitalgh/palladium-oss/internal/report/postgres"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	serviceequipmentpostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as every other
// repository test file in this codebase (see e.g.
// internal/serviceequipment/postgres/service_equipment_test.go). Report
// tests need the deepest fixture surface of any package so far: each of
// the three reports joins across several other domains' own tables, so
// every fixture here is built through that domain's own real repository
// (see each createTest* helper's own doc comment) rather than raw SQL,
// the same reasoning those other packages' fixtures already give.
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
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// createTestCustomer creates a real Customer row through
// internal/customer/postgres.
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

// createTestContact creates a real Contact row for customerID through
// internal/contact/postgres.
func createTestContact(t *testing.T, ctx context.Context, q database.Querier, customerID uuid.UUID) contact.Contact {
	t.Helper()

	repo := contactpostgres.NewContactRepository(q, clock.New(), id.New())
	c, err := repo.Create(ctx, contact.Contact{
		CustomerID: customerID,
		Name:       "Fixture Contact " + uuid.NewString(),
		Role:       contact.ContactRolePrimary,
		Email:      "fixture-" + uuid.NewString() + "@example.com",
		Status:     contact.ContactStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create contact: %v", err)
	}
	return c
}

// createTestDevice creates a real Device row (and the fixture Device
// Manufacturer/Model chain it requires) through
// internal/inventory/postgres, mirroring
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

// attachCustomerDevice creates a real, active CustomerDevice placement
// through internal/customerdevice/postgres.
func attachCustomerDevice(t *testing.T, ctx context.Context, q database.Querier, customerID, deviceID uuid.UUID) customerdevice.CustomerDevice {
	t.Helper()

	repo := customerdevicepostgres.NewCustomerDeviceRepository(q, clock.New(), id.New())
	cd, err := repo.Create(ctx, customerdevice.CustomerDevice{CustomerID: customerID, DeviceID: deviceID})
	if err != nil {
		t.Fatalf("fixture: create customer device: %v", err)
	}
	return cd
}

// createTestServiceForCustomer creates a real Service row delivered to
// customerID (and the fixture Location/Catalog/Provider/Product chain
// it requires), the same fixture chain
// internal/serviceequipment/postgres/service_equipment_test.go's own
// createTestService builds, parameterized on an existing Customer so a
// test can put more than one Service under the same Customer.
func createTestServiceForCustomer(t *testing.T, ctx context.Context, q database.Querier, customerID uuid.UUID) domainservice.Service {
	t.Helper()

	locationRepo := locationpostgres.NewLocationRepository(q, clock.New(), id.New())
	l, err := locationRepo.Create(ctx, location.Location{
		CustomerID: customerID,
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

// attachServiceEquipment creates a real, active ServiceEquipment
// assignment through internal/serviceequipment/postgres. Role is
// EquipmentRoleRouter, not ONU/ONT, so no UNIPort needs setting (see
// serviceequipment.ServiceEquipment.Validate's own UNIPort rule, which
// only applies to ONU/ONT).
func attachServiceEquipment(t *testing.T, ctx context.Context, q database.Querier, serviceID, deviceID uuid.UUID) serviceequipment.ServiceEquipment {
	t.Helper()

	repo := serviceequipmentpostgres.NewServiceEquipmentRepository(q, clock.New(), id.New())
	se, err := repo.Create(ctx, serviceequipment.ServiceEquipment{
		ServiceID: serviceID,
		DeviceID:  deviceID,
		Role:      serviceequipment.EquipmentRoleRouter,
	})
	if err != nil {
		t.Fatalf("fixture: create service equipment: %v", err)
	}
	return se
}

func TestCustomersWithContactsIncludesCustomerWithNoContacts(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	repo := postgres.NewRepository(q)

	rows, err := repo.CustomersWithContacts(ctx)
	if err != nil {
		t.Fatalf("CustomersWithContacts() = %v", err)
	}

	found := false
	for _, row := range rows {
		if row.CustomerID != c.ID {
			continue
		}
		found = true
		if row.ContactID != uuid.Nil {
			t.Errorf("ContactID = %v, want uuid.Nil for a customer with no contacts", row.ContactID)
		}
		if row.ContactName != "" {
			t.Errorf("ContactName = %q, want empty for a customer with no contacts", row.ContactName)
		}
	}
	if !found {
		t.Fatalf("no row for customer %v, want exactly one (with blank contact fields)", c.ID)
	}
}

func TestCustomersWithContactsListsEveryContact(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	contact1 := createTestContact(t, ctx, q, c.ID)
	contact2 := createTestContact(t, ctx, q, c.ID)
	repo := postgres.NewRepository(q)

	rows, err := repo.CustomersWithContacts(ctx)
	if err != nil {
		t.Fatalf("CustomersWithContacts() = %v", err)
	}

	gotContactIDs := map[uuid.UUID]bool{}
	for _, row := range rows {
		if row.CustomerID == c.ID {
			gotContactIDs[row.ContactID] = true
		}
	}
	if !gotContactIDs[contact1.ID] || !gotContactIDs[contact2.ID] {
		t.Errorf("contact IDs for customer %v = %v, want both %v and %v present", c.ID, gotContactIDs, contact1.ID, contact2.ID)
	}
}

func TestDevicesResolvesAssignedCustomerViaPlacement(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	d := createTestDevice(t, ctx, q)
	attachCustomerDevice(t, ctx, q, c.ID, d.ID)
	repo := postgres.NewRepository(q)

	rows, err := repo.Devices(ctx)
	if err != nil {
		t.Fatalf("Devices() = %v", err)
	}

	row := findDeviceRow(t, rows, d.ID)
	if row.AssignedCustomerID != c.ID {
		t.Errorf("AssignedCustomerID = %v, want %v", row.AssignedCustomerID, c.ID)
	}
	if row.AssignedCustomerName != c.Name {
		t.Errorf("AssignedCustomerName = %q, want %q", row.AssignedCustomerName, c.Name)
	}
}

func TestDevicesResolvesAssignedCustomerViaServiceEquipment(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)
	s := createTestServiceForCustomer(t, ctx, q, c.ID)
	d := createTestDevice(t, ctx, q)
	attachServiceEquipment(t, ctx, q, s.ID, d.ID)
	repo := postgres.NewRepository(q)

	rows, err := repo.Devices(ctx)
	if err != nil {
		t.Fatalf("Devices() = %v", err)
	}

	row := findDeviceRow(t, rows, d.ID)
	if row.AssignedCustomerID != c.ID {
		t.Errorf("AssignedCustomerID = %v, want %v", row.AssignedCustomerID, c.ID)
	}
	if row.AssignedCustomerName != c.Name {
		t.Errorf("AssignedCustomerName = %q, want %q", row.AssignedCustomerName, c.Name)
	}
}

func TestDevicesLeavesAssignedCustomerBlankWhenUnassigned(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	repo := postgres.NewRepository(q)

	rows, err := repo.Devices(ctx)
	if err != nil {
		t.Fatalf("Devices() = %v", err)
	}

	row := findDeviceRow(t, rows, d.ID)
	if row.AssignedCustomerID != uuid.Nil {
		t.Errorf("AssignedCustomerID = %v, want uuid.Nil for an unassigned device", row.AssignedCustomerID)
	}
	if row.AssignedCustomerName != "" {
		t.Errorf("AssignedCustomerName = %q, want empty for an unassigned device", row.AssignedCustomerName)
	}
}

// TestCustomersWithDevicesReturnsOneRowPerRelationship is the direct
// proof behind report.CustomerDeviceRow's own doc comment: a Customer
// with both an active Placement and an active Service Equipment
// assignment gets one row per relationship, not one collapsed row per
// Device the way Devices' single AssignedCustomer column does.
func TestCustomersWithDevicesReturnsOneRowPerRelationship(t *testing.T) {
	q, ctx := newTestQuerier(t)
	c := createTestCustomer(t, ctx, q)

	placedDevice := createTestDevice(t, ctx, q)
	attachCustomerDevice(t, ctx, q, c.ID, placedDevice.ID)

	s := createTestServiceForCustomer(t, ctx, q, c.ID)
	deliveredDevice := createTestDevice(t, ctx, q)
	attachServiceEquipment(t, ctx, q, s.ID, deliveredDevice.ID)

	repo := postgres.NewRepository(q)
	rows, err := repo.CustomersWithDevices(ctx)
	if err != nil {
		t.Fatalf("CustomersWithDevices() = %v", err)
	}

	var placementRelationship, serviceEquipmentRelationship string
	placementFound, serviceEquipmentFound := false, false
	for _, row := range rows {
		if row.CustomerID != c.ID {
			continue
		}
		switch row.DeviceID {
		case placedDevice.ID:
			placementFound = true
			placementRelationship = row.Relationship
		case deliveredDevice.ID:
			serviceEquipmentFound = true
			serviceEquipmentRelationship = row.Relationship
			if row.LocationName == "" {
				t.Error("LocationName is blank for a Service Equipment row, want the Service's Location name")
			}
		}
	}

	if !placementFound {
		t.Fatalf("no row for customer %v / placed device %v", c.ID, placedDevice.ID)
	}
	if placementRelationship != "Placement" {
		t.Errorf("Relationship = %q, want %q", placementRelationship, "Placement")
	}
	if !serviceEquipmentFound {
		t.Fatalf("no row for customer %v / delivered device %v", c.ID, deliveredDevice.ID)
	}
	if serviceEquipmentRelationship != "Service Equipment" {
		t.Errorf("Relationship = %q, want %q", serviceEquipmentRelationship, "Service Equipment")
	}
}

func findDeviceRow(t *testing.T, rows []report.DeviceRow, deviceID uuid.UUID) report.DeviceRow {
	t.Helper()
	for _, row := range rows {
		if row.DeviceID == deviceID {
			return row
		}
	}
	t.Fatalf("no row for device %v", deviceID)
	return report.DeviceRow{}
}
