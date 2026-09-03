package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeDeviceRepository is an in-memory inventory.DeviceRepository. See
// fakeSiteRepository (site_service_test.go) for why this exists instead of
// a real database.
type fakeDeviceRepository struct {
	byID         map[uuid.UUID]inventory.Device
	createCalled bool
	updateCalled bool
}

func newFakeDeviceRepository(devices ...inventory.Device) *fakeDeviceRepository {
	f := &fakeDeviceRepository{byID: make(map[uuid.UUID]inventory.Device)}
	for _, d := range devices {
		f.byID[d.ID] = d
	}
	return f
}

func (f *fakeDeviceRepository) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	d, ok := f.byID[id]
	if !ok {
		return inventory.Device{}, apperror.NotFound("device not found")
	}
	return d, nil
}

func (f *fakeDeviceRepository) List(_ context.Context) ([]inventory.Device, error) {
	devices := make([]inventory.Device, 0, len(f.byID))
	for _, d := range f.byID {
		devices = append(devices, d)
	}
	return devices, nil
}

func (f *fakeDeviceRepository) Create(_ context.Context, device inventory.Device) (inventory.Device, error) {
	f.createCalled = true
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	f.byID[device.ID] = device
	return device, nil
}

func (f *fakeDeviceRepository) Update(_ context.Context, device inventory.Device) (inventory.Device, error) {
	f.updateCalled = true
	if _, ok := f.byID[device.ID]; !ok {
		return inventory.Device{}, apperror.NotFound("device not found")
	}
	f.byID[device.ID] = device
	return device, nil
}

func (f *fakeDeviceRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("device not found")
	}
	delete(f.byID, id)
	return nil
}

var _ inventory.DeviceRepository = (*fakeDeviceRepository)(nil)

func validDevice() inventory.Device {
	return inventory.Device{
		Metadata:     inventory.Metadata{Name: "ONT-Main-01"},
		Manufacturer: "Calix",
		Model:        "716GE",
		SerialNumber: "CXNK00112233",
		Status:       inventory.DeviceStatusInStock,
	}
}

func TestDeviceServiceCreateSucceeds(t *testing.T) {
	repo := newFakeDeviceRepository()
	svc := service.NewDeviceService(repo)

	created, err := svc.Create(context.Background(), validDevice())
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if !repo.createCalled {
		t.Error("repository Create() was never called")
	}
}

func TestDeviceServiceCreateRejectsInvalidDeviceWithoutPersisting(t *testing.T) {
	repo := newFakeDeviceRepository()
	svc := service.NewDeviceService(repo)

	_, err := svc.Create(context.Background(), inventory.Device{}) // no Name, Manufacturer, Model, SerialNumber, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestDeviceServiceUpdateSucceeds(t *testing.T) {
	existing := validDevice()
	existing.ID = uuid.New()
	repo := newFakeDeviceRepository(existing)
	svc := service.NewDeviceService(repo)

	updated := existing
	updated.Name = "ONT-Main-02"

	result, err := svc.Update(context.Background(), updated)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if result.Name != "ONT-Main-02" {
		t.Errorf("Name = %q, want %q", result.Name, "ONT-Main-02")
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestDeviceServiceUpdateRejectsInvalidDeviceWithoutPersisting(t *testing.T) {
	existing := validDevice()
	existing.ID = uuid.New()
	repo := newFakeDeviceRepository(existing)
	svc := service.NewDeviceService(repo)

	invalid := existing
	invalid.Name = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestDeviceServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeDeviceRepository()
	svc := service.NewDeviceService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestDeviceServiceListDelegatesToRepository(t *testing.T) {
	a := validDevice()
	a.ID = uuid.New()
	b := validDevice()
	b.ID = uuid.New()
	repo := newFakeDeviceRepository(a, b)
	svc := service.NewDeviceService(repo)

	devices, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(devices))
	}
}

func TestDeviceServiceDeleteSucceeds(t *testing.T) {
	existing := validDevice()
	existing.ID = uuid.New()
	repo := newFakeDeviceRepository(existing)
	svc := service.NewDeviceService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestDeviceServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeDeviceRepository()
	svc := service.NewDeviceService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
