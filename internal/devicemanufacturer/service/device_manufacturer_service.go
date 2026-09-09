// Package service is the Device Manufacturer domain's business logic
// layer. It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/devicemanufacturer/httpapi), and repositories never validate
// or otherwise reason about business rules (see
// internal/devicemanufacturer/postgres, which trusts its caller) — this
// is where those two responsibilities meet. It mirrors
// internal/oltmodel/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer"
)

// DeviceManufacturerService is the Device Manufacturer domain's business
// logic.
type DeviceManufacturerService struct {
	manufacturers devicemanufacturer.DeviceManufacturerRepository
}

// NewDeviceManufacturerService builds a DeviceManufacturerService.
func NewDeviceManufacturerService(manufacturers devicemanufacturer.DeviceManufacturerRepository) *DeviceManufacturerService {
	return &DeviceManufacturerService{manufacturers: manufacturers}
}

// Get retrieves a DeviceManufacturer by ID.
func (s *DeviceManufacturerService) Get(ctx context.Context, id uuid.UUID) (devicemanufacturer.DeviceManufacturer, error) {
	return s.manufacturers.Get(ctx, id)
}

// List returns every DeviceManufacturer.
func (s *DeviceManufacturerService) List(ctx context.Context) ([]devicemanufacturer.DeviceManufacturer, error) {
	return s.manufacturers.List(ctx)
}

// Create validates m and, if valid, persists it. See
// oltmodel/service.OLTModelService.Create for why validation happens
// here rather than in the handler or repository.
func (s *DeviceManufacturerService) Create(ctx context.Context, m devicemanufacturer.DeviceManufacturer) (devicemanufacturer.DeviceManufacturer, error) {
	if err := m.Validate(); err != nil {
		return devicemanufacturer.DeviceManufacturer{}, err
	}
	return s.manufacturers.Create(ctx, m)
}

// Update validates m and, if valid, persists the change.
func (s *DeviceManufacturerService) Update(ctx context.Context, m devicemanufacturer.DeviceManufacturer) (devicemanufacturer.DeviceManufacturer, error) {
	if err := m.Validate(); err != nil {
		return devicemanufacturer.DeviceManufacturer{}, err
	}
	return s.manufacturers.Update(ctx, m)
}

// Delete removes the DeviceManufacturer identified by id.
func (s *DeviceManufacturerService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.manufacturers.Delete(ctx, id)
}
