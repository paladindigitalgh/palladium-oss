//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	accessinterfacepostgres "github.com/paladindigitalgh/palladium-oss/internal/accessinterface/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	accessnetworkpostgres "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	catalogpostgres "github.com/paladindigitalgh/palladium-oss/internal/catalog/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	locationpostgres "github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	oltpostgres "github.com/paladindigitalgh/palladium-oss/internal/olt/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
	ponportpostgres "github.com/paladindigitalgh/palladium-oss/internal/ponport/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
	providerpostgres "github.com/paladindigitalgh/palladium-oss/internal/provider/postgres"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	serviceequipmentpostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
	serviceprofilepostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/postgres"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/serviceequipment/postgres/service_equipment_test.go.
// AccessAttachment needs the deepest fixture chain in this codebase so
// far: a fixture AccessInterface (itself needing AccessNetwork -> OLT ->
// PONPort) AND a fixture ServiceEquipment (itself needing Service, which
// needs Location -> Customer and Product -> Catalog, plus a Device), all
// sharing this one transaction.
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

// createTestAccessInterface creates a real AccessInterface row (and the
// fixture PONPort/OLT/AccessNetwork chain it requires) through
// internal/accessinterface/postgres and its own dependencies — not
// internal/accessattachment/postgres — so an AccessAttachment fixture
// failure surfaces as a clear failure of one specific layer, not a
// confusing failure somewhere else. This is the one place this package
// imports internal/accessinterface (and everything it depends on) at
// all: the domain model (internal/accessattachment) never does (see its
// package doc comment), only this test, which genuinely needs a real
// access_interfaces row for the foreign key to reference.
func createTestAccessInterface(t *testing.T, ctx context.Context, q database.Querier) accessinterface.AccessInterface {
	t.Helper()

	accessNetworkRepo := accessnetworkpostgres.NewAccessNetworkRepository(q, clock.New(), id.New())
	a, err := accessNetworkRepo.Create(ctx, accessnetwork.AccessNetwork{
		Name:   "Fixture Access Network " + uuid.NewString(),
		Status: accessnetwork.AccessNetworkStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create access network: %v", err)
	}

	oltRepo := oltpostgres.NewOLTRepository(q, clock.New(), id.New())
	o, err := oltRepo.Create(ctx, olt.OLT{
		AccessNetworkID: a.ID,
		Name:            "Fixture OLT " + uuid.NewString(),
		Vendor:          olt.VendorKontron,
	})
	if err != nil {
		t.Fatalf("fixture: create olt: %v", err)
	}

	ponPortRepo := ponportpostgres.NewPONPortRepository(q, clock.New(), id.New())
	p, err := ponPortRepo.Create(ctx, ponport.PONPort{
		OLTID:      o.ID,
		PortNumber: 1,
	})
	if err != nil {
		t.Fatalf("fixture: create pon port: %v", err)
	}

	interfaceRepo := accessinterfacepostgres.NewAccessInterfaceRepository(q, clock.New(), id.New())
	iface, err := interfaceRepo.Create(ctx, accessinterface.AccessInterface{
		PONPortID:  p.ID,
		Technology: accessinterface.TechnologyGPON,
		Name:       "Fixture Interface " + uuid.NewString(),
		Status:     accessinterface.StatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create access interface: %v", err)
	}
	return iface
}

// createTestServiceEquipment creates a real ServiceEquipment row (and
// the fixture Service -> Location/Customer/Product/Catalog chain, plus a
// fixture Device, that it requires) through
// internal/serviceequipment/postgres and its own dependencies, mirroring
// internal/serviceequipment/postgres/service_equipment_test.go's own
// createTestService/createTestDevice fixtures exactly — this test
// genuinely needs a real service_equipment row for the foreign key to
// reference.
func createTestServiceEquipment(t *testing.T, ctx context.Context, q database.Querier) serviceequipment.ServiceEquipment {
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
		CatalogID:  cat.ID,
		ProviderID: pr.ID,
		Name:       "Fixture Product " + uuid.NewString(),
		Category:   product.ProductCategoryInternet,
		Status:     product.ProductStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create product: %v", err)
	}

	profileRepo := serviceprofilepostgres.NewServiceProfileRepository(q, clock.New(), id.New())
	sp, err := profileRepo.Create(ctx, serviceprofile.ServiceProfile{
		Name:   "Fixture Service Profile " + uuid.NewString(),
		Status: serviceprofile.StatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create service profile: %v", err)
	}

	serviceRepo := servicepostgres.NewServiceRepository(q, clock.New(), id.New())
	s, err := serviceRepo.Create(ctx, domainservice.Service{
		LocationID:       l.ID,
		ProductID:        p.ID,
		ServiceProfileID: sp.ID,
		Status:           domainservice.ServiceStatusActive,
	})
	if err != nil {
		t.Fatalf("fixture: create service: %v", err)
	}

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

	equipmentRepo := serviceequipmentpostgres.NewServiceEquipmentRepository(q, clock.New(), id.New())
	e, err := equipmentRepo.Create(ctx, serviceequipment.ServiceEquipment{
		ServiceID: s.ID,
		DeviceID:  d.ID,
		Role:      serviceequipment.EquipmentRoleONU,
	})
	if err != nil {
		t.Fatalf("fixture: create service equipment: %v", err)
	}
	return e
}

func testAccessAttachment(accessInterfaceID, serviceEquipmentID uuid.UUID) accessattachment.AccessAttachment {
	return accessattachment.AccessAttachment{
		AccessInterfaceID:  accessInterfaceID,
		ServiceEquipmentID: serviceEquipmentID,
	}
}

func TestAccessAttachmentRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	installedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, accessattachment.AccessAttachment{
		AccessInterfaceID:  iface.ID,
		ServiceEquipmentID: eq.ID,
		InstalledAt:        &installedAt,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.AccessInterfaceID != iface.ID {
		t.Errorf("AccessInterfaceID = %v, want %v", created.AccessInterfaceID, iface.ID)
	}
	if created.ServiceEquipmentID != eq.ID {
		t.Errorf("ServiceEquipmentID = %v, want %v", created.ServiceEquipmentID, eq.ID)
	}
	if created.InstalledAt == nil || !created.InstalledAt.Equal(installedAt) {
		t.Errorf("InstalledAt = %v, want %v", created.InstalledAt, installedAt)
	}
	if created.RemovedAt != nil {
		t.Errorf("RemovedAt = %v, want nil", created.RemovedAt)
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

func TestAccessAttachmentRepositoryCreateWithoutLifecycleTimestamps(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq.ID))
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

func TestAccessAttachmentRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	a := testAccessAttachment(iface.ID, eq.ID)
	a.ID = bogusID
	a.CreatedAt = bogusTime
	a.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, a)
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

func TestAccessAttachmentRepositoryCreateFailsWhenAccessInterfaceDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testAccessAttachment(uuid.New(), eq.ID)) // access interface does not exist

	assertConflict(t, err)
}

func TestAccessAttachmentRepositoryCreateFailsWhenServiceEquipmentDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, testAccessAttachment(iface.ID, uuid.New())) // service equipment does not exist

	assertConflict(t, err)
}

func TestAccessAttachmentRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.AccessInterfaceID != created.AccessInterfaceID || got.ServiceEquipmentID != created.ServiceEquipmentID {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestAccessAttachmentRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestAccessAttachmentRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq1 := createTestServiceEquipment(t, ctx, q)
	eq2 := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq1.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq2.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	attachments, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]accessattachment.AccessAttachment, len(attachments))
	for _, a := range attachments {
		found[a.ID] = a
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created AccessAttachment")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created AccessAttachment")
	}
	if len(attachments) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(attachments), attachments)
	}
}

func TestAccessAttachmentRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	otherEquipment := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	installedAt := time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)
	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	updated, err := repo.Update(ctx, accessattachment.AccessAttachment{
		ID:                 created.ID,
		AccessInterfaceID:  iface.ID,
		ServiceEquipmentID: otherEquipment.ID,
		InstalledAt:        &installedAt,
		RemovedAt:          &removedAt,
		RemovalReason:      "Customer moved",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.ServiceEquipmentID != otherEquipment.ID {
		t.Errorf("ServiceEquipmentID = %v, want %v (ServiceEquipmentID must be mutable via Update)", updated.ServiceEquipmentID, otherEquipment.ID)
	}
	if updated.RemovalReason != "Customer moved" {
		t.Errorf("RemovalReason = %q, want %q", updated.RemovalReason, "Customer moved")
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

func TestAccessAttachmentRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	ghost := testAccessAttachment(iface.ID, eq.ID)
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestAccessAttachmentRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestAccessAttachmentRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestAccessAttachmentRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	fixedID := uuid.New()
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq.ID)); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	otherEquipment := createTestServiceEquipment(t, ctx, q)
	_, err := repo.Create(ctx, testAccessAttachment(iface.ID, otherEquipment.ID))
	assertConflict(t, err)
}

// TestAccessAttachmentRepositoryGetActiveByServiceEquipmentIDReturnsActiveAttachment
// proves the query goal 4 asks for by name finds the one active
// (RemovedAt == nil) attachment for a ServiceEquipment.
func TestAccessAttachmentRepositoryGetActiveByServiceEquipmentIDReturnsActiveAttachment(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	active, err := repo.GetActiveByServiceEquipmentID(ctx, eq.ID)
	if err != nil {
		t.Fatalf("GetActiveByServiceEquipmentID() = %v", err)
	}
	if active.ID != created.ID {
		t.Errorf("GetActiveByServiceEquipmentID() = %+v, want %+v", active, created)
	}
}

// TestAccessAttachmentRepositoryGetActiveByServiceEquipmentIDNotFoundWhenNoneActive
// proves GetActiveByServiceEquipmentID reports not-found both when
// equipment has never had an attachment, and when its only attachment
// has been removed — "historical moves are allowed" (this milestone's
// goal 2) means a removed attachment must not still count as active.
func TestAccessAttachmentRepositoryGetActiveByServiceEquipmentIDNotFoundWhenNoneActive(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	repo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())

	_, err := repo.GetActiveByServiceEquipmentID(ctx, eq.ID)
	assertNotFound(t, err)

	created, err := repo.Create(ctx, testAccessAttachment(iface.ID, eq.ID))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	created.RemovedAt = &removedAt
	if _, err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	_, err = repo.GetActiveByServiceEquipmentID(ctx, eq.ID)
	assertNotFound(t, err)
}

// TestAccessInterfaceRepositoryDeleteBlockedByExistingAccessAttachment
// lives here, not in internal/accessinterface/postgres, so that
// package's existing test files stay untouched — the same reasoning
// internal/serviceequipment/postgres/service_equipment_test.go already
// documents for why its equivalent tests live with the child, not the
// parent.
func TestAccessInterfaceRepositoryDeleteBlockedByExistingAccessAttachment(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	attachmentRepo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())
	if _, err := attachmentRepo.Create(ctx, testAccessAttachment(iface.ID, eq.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	interfaceRepo := accessinterfacepostgres.NewAccessInterfaceRepository(q, clock.New(), id.New())

	err := interfaceRepo.Delete(ctx, iface.ID)

	assertConflict(t, err)
}

// TestServiceEquipmentRepositoryDeleteBlockedByExistingAccessAttachment
// is TestAccessInterfaceRepositoryDeleteBlockedByExistingAccessAttachment's
// counterpart for the other foreign key.
func TestServiceEquipmentRepositoryDeleteBlockedByExistingAccessAttachment(t *testing.T) {
	q, ctx := newTestQuerier(t)
	iface := createTestAccessInterface(t, ctx, q)
	eq := createTestServiceEquipment(t, ctx, q)
	attachmentRepo := postgres.NewAccessAttachmentRepository(q, clock.New(), id.New())
	if _, err := attachmentRepo.Create(ctx, testAccessAttachment(iface.ID, eq.ID)); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	equipmentRepo := serviceequipmentpostgres.NewServiceEquipmentRepository(q, clock.New(), id.New())

	err := equipmentRepo.Delete(ctx, eq.ID)

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
