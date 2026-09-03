package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// BuildingService is the Inventory domain's business logic for Buildings.
// See SiteService's doc comment for why this depends only on
// inventory.BuildingRepository, not clock.Clock.
type BuildingService struct {
	buildings inventory.BuildingRepository
}

// NewBuildingService builds a BuildingService.
func NewBuildingService(buildings inventory.BuildingRepository) *BuildingService {
	return &BuildingService{buildings: buildings}
}

// Get retrieves a Building by ID.
func (s *BuildingService) Get(ctx context.Context, id uuid.UUID) (inventory.Building, error) {
	return s.buildings.Get(ctx, id)
}

// List returns every Building.
func (s *BuildingService) List(ctx context.Context) ([]inventory.Building, error) {
	return s.buildings.List(ctx)
}

// Create validates building and, if valid, persists it. See
// SiteService.Create for why validation happens here rather than
// elsewhere.
func (s *BuildingService) Create(ctx context.Context, building inventory.Building) (inventory.Building, error) {
	if err := building.Validate(); err != nil {
		return inventory.Building{}, err
	}
	return s.buildings.Create(ctx, building)
}

// Update validates building and, if valid, persists the change. See
// SiteService.Create for why validation happens here rather than
// elsewhere.
func (s *BuildingService) Update(ctx context.Context, building inventory.Building) (inventory.Building, error) {
	if err := building.Validate(); err != nil {
		return inventory.Building{}, err
	}
	return s.buildings.Update(ctx, building)
}

// Delete removes the Building identified by id.
func (s *BuildingService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.buildings.Delete(ctx, id)
}
