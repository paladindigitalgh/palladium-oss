package inventory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// The stub repositories below have no SQL implementation to test yet — that
// is intentionally out of scope for this milestone (see repository.go).
// They exist solely to prove each interface is satisfiable with a sane,
// consistent method shape. The var block's compile-time assertions are the
// actual check: this file fails to build if a repository interface and its
// stub ever drift apart.

type stubSiteRepository struct{}

func (stubSiteRepository) Get(context.Context, uuid.UUID) (inventory.Site, error) {
	return inventory.Site{}, nil
}
func (stubSiteRepository) List(context.Context) ([]inventory.Site, error) { return nil, nil }
func (stubSiteRepository) Create(_ context.Context, s inventory.Site) (inventory.Site, error) {
	return s, nil
}
func (stubSiteRepository) Update(_ context.Context, s inventory.Site) (inventory.Site, error) {
	return s, nil
}
func (stubSiteRepository) Delete(context.Context, uuid.UUID) error { return nil }

type stubBuildingRepository struct{}

func (stubBuildingRepository) Get(context.Context, uuid.UUID) (inventory.Building, error) {
	return inventory.Building{}, nil
}
func (stubBuildingRepository) List(context.Context) ([]inventory.Building, error) { return nil, nil }
func (stubBuildingRepository) Create(_ context.Context, b inventory.Building) (inventory.Building, error) {
	return b, nil
}
func (stubBuildingRepository) Update(_ context.Context, b inventory.Building) (inventory.Building, error) {
	return b, nil
}
func (stubBuildingRepository) Delete(context.Context, uuid.UUID) error { return nil }

type stubRoomRepository struct{}

func (stubRoomRepository) Get(context.Context, uuid.UUID) (inventory.Room, error) {
	return inventory.Room{}, nil
}
func (stubRoomRepository) List(context.Context) ([]inventory.Room, error) { return nil, nil }
func (stubRoomRepository) Create(_ context.Context, r inventory.Room) (inventory.Room, error) {
	return r, nil
}
func (stubRoomRepository) Update(_ context.Context, r inventory.Room) (inventory.Room, error) {
	return r, nil
}
func (stubRoomRepository) Delete(context.Context, uuid.UUID) error { return nil }

type stubRackRepository struct{}

func (stubRackRepository) Get(context.Context, uuid.UUID) (inventory.Rack, error) {
	return inventory.Rack{}, nil
}
func (stubRackRepository) List(context.Context) ([]inventory.Rack, error) { return nil, nil }
func (stubRackRepository) Create(_ context.Context, r inventory.Rack) (inventory.Rack, error) {
	return r, nil
}
func (stubRackRepository) Update(_ context.Context, r inventory.Rack) (inventory.Rack, error) {
	return r, nil
}
func (stubRackRepository) Delete(context.Context, uuid.UUID) error { return nil }

type stubDeviceRepository struct{}

func (stubDeviceRepository) Get(context.Context, uuid.UUID) (inventory.Device, error) {
	return inventory.Device{}, nil
}
func (stubDeviceRepository) List(context.Context) ([]inventory.Device, error) { return nil, nil }
func (stubDeviceRepository) Create(_ context.Context, d inventory.Device) (inventory.Device, error) {
	return d, nil
}
func (stubDeviceRepository) Update(_ context.Context, d inventory.Device) (inventory.Device, error) {
	return d, nil
}
func (stubDeviceRepository) Delete(context.Context, uuid.UUID) error { return nil }

var (
	_ inventory.SiteRepository     = (*stubSiteRepository)(nil)
	_ inventory.BuildingRepository = (*stubBuildingRepository)(nil)
	_ inventory.RoomRepository     = (*stubRoomRepository)(nil)
	_ inventory.RackRepository     = (*stubRackRepository)(nil)
	_ inventory.DeviceRepository   = (*stubDeviceRepository)(nil)
)

func TestRepositoryInterfacesAreSatisfiable(t *testing.T) {
	// The compile-time assertions above are the real test: if this package
	// builds, every repository interface has the intended Get/List/Create/
	// Update/Delete shape. This test exists so `go test` reports that
	// check explicitly instead of the file silently containing no tests.
}
