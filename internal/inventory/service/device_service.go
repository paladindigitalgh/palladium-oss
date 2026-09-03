package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// DeviceService is the Inventory domain's business logic for Devices. It
// follows SiteService exactly (see that file for why no clock.Clock is
// needed here either — DeviceRepository stamps its own timestamps; see
// internal/inventory/postgres/device.go).
type DeviceService struct {
	devices inventory.DeviceRepository
}

// NewDeviceService builds a DeviceService.
func NewDeviceService(devices inventory.DeviceRepository) *DeviceService {
	return &DeviceService{devices: devices}
}

// Get retrieves a Device by ID.
func (s *DeviceService) Get(ctx context.Context, id uuid.UUID) (inventory.Device, error) {
	return s.devices.Get(ctx, id)
}

// List returns every Device.
func (s *DeviceService) List(ctx context.Context) ([]inventory.Device, error) {
	return s.devices.List(ctx)
}

// Create validates device and, if valid, persists it. See SiteService.Create
// for why validation happens here rather than in the repository or the HTTP
// handler.
func (s *DeviceService) Create(ctx context.Context, device inventory.Device) (inventory.Device, error) {
	if err := device.Validate(); err != nil {
		return inventory.Device{}, err
	}
	return s.devices.Create(ctx, device)
}

// Update validates device and, if valid, persists the change. See
// SiteService.Update for why validation happens here rather than elsewhere.
func (s *DeviceService) Update(ctx context.Context, device inventory.Device) (inventory.Device, error) {
	if err := device.Validate(); err != nil {
		return inventory.Device{}, err
	}
	return s.devices.Update(ctx, device)
}

// Delete removes the Device identified by id.
func (s *DeviceService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.devices.Delete(ctx, id)
}
