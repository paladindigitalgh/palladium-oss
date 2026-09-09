//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// validDevice returns a Device that satisfies every column with a NOT NULL
// constraint (device_model_id, serial_number, status), so tests below
// only need to override the field(s) they actually care about. Mirrors
// validDevice in internal/inventory/validate_test.go, except it needs a
// real DeviceModel fixture (see createTestDeviceModel in
// testing_test.go): device_model_id is a foreign key, unlike the plain
// strings it replaced.
func validDevice(t *testing.T, ctx context.Context, q database.Querier, name string) inventory.Device {
	t.Helper()

	model := createTestDeviceModel(t, ctx, q)
	return inventory.Device{
		Metadata:      inventory.Metadata{Name: name},
		DeviceModelID: model.ID,
		SerialNumber:  "SN-" + uuid.NewString(),
		Status:        inventory.DeviceStatusUnused,
	}
}

func TestDeviceRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	rack := createTestRack(t, ctx, q)
	rackID := rack.ID
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	device := validDevice(t, ctx, q, "Switch 1")
	device.Description = "24-port"
	device.AssetTag = "AT-001"
	device.RackID = &rackID

	created, err := repo.Create(ctx, device)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.RackID == nil || *created.RackID != rack.ID {
		t.Errorf("RackID = %v, want %v", created.RackID, rack.ID)
	}
	if created.DeviceModelID != device.DeviceModelID {
		t.Errorf("DeviceModelID = %v, want %v", created.DeviceModelID, device.DeviceModelID)
	}
	if created.SerialNumber != device.SerialNumber {
		t.Errorf("SerialNumber = %q, want %q", created.SerialNumber, device.SerialNumber)
	}
	if created.AssetTag != "AT-001" {
		t.Errorf("AssetTag = %q, want %q", created.AssetTag, "AT-001")
	}
	if created.Status != inventory.DeviceStatusUnused {
		t.Errorf("Status = %q, want %q", created.Status, inventory.DeviceStatusUnused)
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

// TestDeviceRepositoryCreateWithoutRack exercises the nullable
// relationship: a Device can be ordered, received, and stored before it is
// ever racked (see inventory.Device in internal/inventory/model.go).
func TestDeviceRepositoryCreateWithoutRack(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, validDevice(t, ctx, q, "Spare Switch"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.RackID != nil {
		t.Errorf("RackID = %v, want nil", created.RackID)
	}
	if created.AssetTag != "" {
		t.Errorf("AssetTag = %q, want empty (optional, unset)", created.AssetTag)
	}
}

func TestDeviceRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	device := validDevice(t, ctx, q, "Edge Device")
	device.ID = bogusID
	device.CreatedAt = bogusTime
	device.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, device)
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

func TestDeviceRepositoryCreateFailsWhenRackDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	bogusRackID := uuid.New() // does not exist
	device := validDevice(t, ctx, q, "Orphan Device")
	device.RackID = &bogusRackID

	_, err := repo.Create(ctx, device)

	assertConflict(t, err)
}

func TestDeviceRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, validDevice(t, ctx, q, "Device A"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.SerialNumber != created.SerialNumber || got.Status != created.Status {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
	if got.RackID != nil {
		t.Errorf("RackID = %v, want nil", got.RackID)
	}
}

func TestDeviceRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestDeviceRepositoryGetBySerialNumber(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, validDevice(t, ctx, q, "Device A"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.GetBySerialNumber(ctx, created.SerialNumber)
	if err != nil {
		t.Fatalf("GetBySerialNumber() = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetBySerialNumber() ID = %v, want %v", got.ID, created.ID)
	}
}

func TestDeviceRepositoryGetBySerialNumberNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	_, err := repo.GetBySerialNumber(ctx, "does-not-exist")

	assertNotFound(t, err)
}

func TestDeviceRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, validDevice(t, ctx, q, "Alpha Device"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, validDevice(t, ctx, q, "Beta Device"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	devices, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]inventory.Device, len(devices))
	for _, d := range devices {
		found[d.ID] = d
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created device")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created device")
	}

	if len(devices) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(devices), devices)
	}
	if devices[0].Name != "Alpha Device" || devices[1].Name != "Beta Device" {
		t.Errorf("List() order = [%q, %q], want [Alpha Device, Beta Device]", devices[0].Name, devices[1].Name)
	}
}

// TestDeviceRepositoryUpdateRacksAndUnracks exercises RackID moving in both
// directions, along with the Device-specific fields, mirroring
// TestRackRepositoryUpdateInstallsAndUninstalls.
func TestDeviceRepositoryUpdateRacksAndUnracks(t *testing.T) {
	q, ctx := newTestQuerier(t)
	rack := createTestRack(t, ctx, q)
	rackID := rack.ID
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, validDevice(t, ctx, q, "Mobile Device"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	racked := created
	racked.RackID = &rackID
	racked.Status = inventory.DeviceStatusActive

	updated, err := repo.Update(ctx, racked)
	if err != nil {
		t.Fatalf("Update() (rack) = %v", err)
	}
	if updated.RackID == nil || *updated.RackID != rack.ID {
		t.Errorf("RackID = %v, want %v", updated.RackID, rack.ID)
	}
	if updated.Status != inventory.DeviceStatusActive {
		t.Errorf("Status = %q, want %q", updated.Status, inventory.DeviceStatusActive)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}

	unracked := updated
	unracked.RackID = nil
	unracked.Status = inventory.DeviceStatusUnused

	final, err := repo.Update(ctx, unracked)
	if err != nil {
		t.Fatalf("Update() (unrack) = %v", err)
	}
	if final.RackID != nil {
		t.Errorf("RackID = %v, want nil", final.RackID)
	}
}

func TestDeviceRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	ghost := validDevice(t, ctx, q, "Ghost")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestDeviceRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	fixedID := uuid.New()
	repo := postgres.NewDeviceRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, validDevice(t, ctx, q, "First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, validDevice(t, ctx, q, "Second"))
	assertConflict(t, err)
}

// TestRackRepositoryDeleteBlockedByExistingDevice mirrors
// TestRoomRepositoryDeleteBlockedByExistingRack in rack_test.go, at the
// bottom of the hierarchy. As there, the block only applies when the
// Device is actually racked (non-nil RackID).
func TestRackRepositoryDeleteBlockedByExistingDevice(t *testing.T) {
	q, ctx := newTestQuerier(t)
	rack := createTestRack(t, ctx, q)
	rackID := rack.ID
	deviceRepo := postgres.NewDeviceRepository(q, clock.New(), id.New())

	device := validDevice(t, ctx, q, "Blocking Device")
	device.RackID = &rackID
	if _, err := deviceRepo.Create(ctx, device); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	rackRepo := postgres.NewRackRepository(q, clock.New(), id.New())

	err := rackRepo.Delete(ctx, rack.ID)

	assertConflict(t, err)
}
