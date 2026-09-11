//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer"
	devicemanufacturerpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemodel"
	devicemodelpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemodel/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	oltpostgres "github.com/paladindigitalgh/palladium-oss/internal/olt/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	oltmodelpostgres "github.com/paladindigitalgh/palladium-oss/internal/oltmodel/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/ponport/postgres/pon_port_test.go. Every OnuAuthorization test
// needs both a fixture Device (itself needing a DeviceModel ->
// DeviceManufacturer) and a fixture OLT, to satisfy the required
// DeviceID/OLTID foreign keys.
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

// createTestOLT creates a real OLT row, mirroring
// internal/ponport/postgres/pon_port_test.go's own createTestOLT
// exactly.
func createTestOLT(t *testing.T, ctx context.Context, q database.Querier) olt.OLT {
	t.Helper()

	oltModelRepo := oltmodelpostgres.NewOLTModelRepository(q, clock.New(), id.New())
	m, err := oltModelRepo.Create(ctx, oltmodel.OLTModel{
		Vendor:       oltmodel.VendorKontron,
		Name:         "Fixture Model " + uuid.NewString(),
		PONPortCount: 16,
	})
	if err != nil {
		t.Fatalf("fixture: create olt model: %v", err)
	}

	oltRepo := oltpostgres.NewOLTRepository(q, clock.New(), id.New())
	o, err := oltRepo.Create(ctx, olt.OLT{
		Name:       "Fixture OLT " + uuid.NewString(),
		OLTModelID: m.ID,
	})
	if err != nil {
		t.Fatalf("fixture: create olt: %v", err)
	}
	return o
}

// createTestDevice creates a real Device row (and the fixture
// DeviceModel/DeviceManufacturer it requires), mirroring
// internal/inventory/postgres/testing_test.go's own
// createTestDeviceModel plus a plain Device Create.
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

func TestOnuAuthorizationRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewOnuAuthorizationRepository(q, clock.New(), id.New())

	authorizedAt := time.Now().UTC().Truncate(time.Microsecond)
	created, err := repo.Create(ctx, onuauthorization.OnuAuthorization{
		DeviceID:     d.ID,
		OLTID:        o.ID,
		Interface:    "xgs/1/3",
		AuthorizedAt: authorizedAt,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.DeviceID != d.ID {
		t.Errorf("DeviceID = %v, want %v", created.DeviceID, d.ID)
	}
	if created.OLTID != o.ID {
		t.Errorf("OLTID = %v, want %v", created.OLTID, o.ID)
	}
	if created.Interface != "xgs/1/3" {
		t.Errorf("Interface = %q, want %q", created.Interface, "xgs/1/3")
	}
	if !created.AuthorizedAt.Equal(authorizedAt) {
		t.Errorf("AuthorizedAt = %v, want %v", created.AuthorizedAt, authorizedAt)
	}
	if created.DeauthorizedAt != nil {
		t.Errorf("DeauthorizedAt = %v, want nil", created.DeauthorizedAt)
	}
	if !created.Active() {
		t.Error("Active() = false, want true for a newly created authorization")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

func TestOnuAuthorizationRepositoryCreateRejectsUnknownDeviceID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewOnuAuthorizationRepository(q, clock.New(), id.New())

	_, err := repo.Create(ctx, onuauthorization.OnuAuthorization{
		DeviceID:     uuid.New(),
		OLTID:        o.ID,
		Interface:    "xgs/1/3",
		AuthorizedAt: time.Now(),
	})
	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestOnuAuthorizationRepositoryGetActiveByDeviceID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewOnuAuthorizationRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, onuauthorization.OnuAuthorization{
		DeviceID:     d.ID,
		OLTID:        o.ID,
		Interface:    "xgs/1/3",
		AuthorizedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("fixture: Create() = %v", err)
	}

	got, err := repo.GetActiveByDeviceID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetActiveByDeviceID() = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %v, want %v", got.ID, created.ID)
	}
}

func TestOnuAuthorizationRepositoryGetActiveByDeviceIDReturnsNotFoundWhenNoneActive(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewOnuAuthorizationRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, onuauthorization.OnuAuthorization{
		DeviceID:     d.ID,
		OLTID:        o.ID,
		Interface:    "xgs/1/3",
		AuthorizedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("fixture: Create() = %v", err)
	}
	now := time.Now()
	created.DeauthorizedAt = &now
	if _, err := repo.Update(ctx, created); err != nil {
		t.Fatalf("fixture: Update() = %v", err)
	}

	_, err = repo.GetActiveByDeviceID(ctx, d.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestOnuAuthorizationRepositoryGetActiveByDeviceIDReturnsNotFoundForUnknownDevice(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewOnuAuthorizationRepository(q, clock.New(), id.New())

	_, err := repo.GetActiveByDeviceID(ctx, uuid.New())
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestOnuAuthorizationRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewOnuAuthorizationRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, onuauthorization.OnuAuthorization{
		DeviceID:     d.ID,
		OLTID:        o.ID,
		Interface:    "xgs/1/3",
		AuthorizedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("fixture: Create() = %v", err)
	}

	deauthorizedAt := time.Now().UTC().Truncate(time.Microsecond)
	created.DeauthorizedAt = &deauthorizedAt
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.DeauthorizedAt == nil || !updated.DeauthorizedAt.Equal(deauthorizedAt) {
		t.Errorf("DeauthorizedAt = %v, want %v", updated.DeauthorizedAt, deauthorizedAt)
	}
	if updated.Active() {
		t.Error("Active() = true, want false once DeauthorizedAt is set")
	}
}

func TestOnuAuthorizationRepositoryUpdateReturnsNotFoundForUnknownID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	d := createTestDevice(t, ctx, q)
	o := createTestOLT(t, ctx, q)
	repo := postgres.NewOnuAuthorizationRepository(q, clock.New(), id.New())

	_, err := repo.Update(ctx, onuauthorization.OnuAuthorization{
		ID: uuid.New(), DeviceID: d.ID, OLTID: o.ID, Interface: "xgs/1/3", AuthorizedAt: time.Now(),
	})
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
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
