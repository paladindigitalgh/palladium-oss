// Package service is the OLT Model domain's business logic layer. It
// sits between the HTTP layer and the repository layer: HTTP handlers
// never call a repository directly (see internal/oltmodel/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/oltmodel/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/provider/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
)

// OLTModelService is the OLT Model domain's business logic.
type OLTModelService struct {
	models oltmodel.OLTModelRepository
}

// NewOLTModelService builds an OLTModelService.
func NewOLTModelService(models oltmodel.OLTModelRepository) *OLTModelService {
	return &OLTModelService{models: models}
}

// Get retrieves an OLTModel by ID.
func (s *OLTModelService) Get(ctx context.Context, id uuid.UUID) (oltmodel.OLTModel, error) {
	return s.models.Get(ctx, id)
}

// List returns every OLTModel.
func (s *OLTModelService) List(ctx context.Context) ([]oltmodel.OLTModel, error) {
	return s.models.List(ctx)
}

// Create validates m and, if valid, persists it. See
// provider/service.ProviderService.Create for why validation happens
// here rather than in the handler or repository.
func (s *OLTModelService) Create(ctx context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	if err := m.Validate(); err != nil {
		return oltmodel.OLTModel{}, err
	}
	return s.models.Create(ctx, m)
}

// Update validates m and, if valid, persists the change.
func (s *OLTModelService) Update(ctx context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	if err := m.Validate(); err != nil {
		return oltmodel.OLTModel{}, err
	}
	return s.models.Update(ctx, m)
}

// Delete removes the OLTModel identified by id.
func (s *OLTModelService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.models.Delete(ctx, id)
}
