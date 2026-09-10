// Package service is the Device Model domain's business logic layer. It
// sits between the HTTP layer and the repository layer: HTTP handlers
// never call a repository directly (see internal/devicemodel/httpapi),
// and repositories never validate or otherwise reason about business
// rules (see internal/devicemodel/postgres, which trusts its caller) —
// this is where those two responsibilities meet. It mirrors
// internal/oltmodel/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/devicemodel"
)

// DeviceModelService is the Device Model domain's business logic.
type DeviceModelService struct {
	models devicemodel.DeviceModelRepository
}

// NewDeviceModelService builds a DeviceModelService.
func NewDeviceModelService(models devicemodel.DeviceModelRepository) *DeviceModelService {
	return &DeviceModelService{models: models}
}

// Get retrieves a DeviceModel by ID.
func (s *DeviceModelService) Get(ctx context.Context, id uuid.UUID) (devicemodel.DeviceModel, error) {
	return s.models.Get(ctx, id)
}

// List returns every DeviceModel.
func (s *DeviceModelService) List(ctx context.Context) ([]devicemodel.DeviceModel, error) {
	return s.models.List(ctx)
}

// Create validates m and, if valid, persists it. See
// oltmodel/service.OLTModelService.Create for why validation happens
// here rather than in the handler or repository.
func (s *DeviceModelService) Create(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error) {
	if err := m.Validate(); err != nil {
		return devicemodel.DeviceModel{}, err
	}
	return s.models.Create(ctx, m)
}

// Update validates m and, if valid, persists the change.
func (s *DeviceModelService) Update(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error) {
	if err := m.Validate(); err != nil {
		return devicemodel.DeviceModel{}, err
	}
	return s.models.Update(ctx, m)
}

// Delete removes the DeviceModel identified by id.
func (s *DeviceModelService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.models.Delete(ctx, id)
}

// SetDefault sets or clears the DeviceModel identified by id as the one
// New Device's Model picker pre-selects once that Model's own
// Manufacturer is chosen (see DeviceModel.IsDefault's own doc comment).
// No further validation applies beyond id actually existing -- true or
// false, this is always a legal state for any DeviceModel.
func (s *DeviceModelService) SetDefault(ctx context.Context, id uuid.UUID, isDefault bool) error {
	return s.models.SetDefault(ctx, id, isDefault)
}
