package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// RackService is the Inventory domain's business logic for Racks. See
// SiteService's doc comment for why this depends only on
// inventory.RackRepository, not clock.Clock.
type RackService struct {
	racks inventory.RackRepository
}

// NewRackService builds a RackService.
func NewRackService(racks inventory.RackRepository) *RackService {
	return &RackService{racks: racks}
}

// Get retrieves a Rack by ID.
func (s *RackService) Get(ctx context.Context, id uuid.UUID) (inventory.Rack, error) {
	return s.racks.Get(ctx, id)
}

// List returns every Rack.
func (s *RackService) List(ctx context.Context) ([]inventory.Rack, error) {
	return s.racks.List(ctx)
}

// Create validates rack and, if valid, persists it. See SiteService.Create
// for why validation happens here rather than elsewhere.
func (s *RackService) Create(ctx context.Context, rack inventory.Rack) (inventory.Rack, error) {
	if err := rack.Validate(); err != nil {
		return inventory.Rack{}, err
	}
	return s.racks.Create(ctx, rack)
}

// Update validates rack and, if valid, persists the change. See
// SiteService.Create for why validation happens here rather than
// elsewhere.
func (s *RackService) Update(ctx context.Context, rack inventory.Rack) (inventory.Rack, error) {
	if err := rack.Validate(); err != nil {
		return inventory.Rack{}, err
	}
	return s.racks.Update(ctx, rack)
}

// Delete removes the Rack identified by id.
func (s *RackService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.racks.Delete(ctx, id)
}
