// Package service is the Inventory domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/inventory/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/inventory/postgres, which trusts its caller) — this is
// where those two responsibilities meet.
//
// Only Site is implemented here. Building, Room, Rack, and Device follow
// the same pattern once their own HTTP endpoints exist.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// SiteService is the Inventory domain's business logic for Sites.
//
// It depends only on inventory.SiteRepository — not clock.Clock, even
// though every repository in this codebase takes one. Timestamps are
// already the repository's responsibility (SiteRepository stamps
// CreatedAt/UpdatedAt itself via its own injected clock.Clock; see
// internal/inventory/postgres/site.go), and this service has no business
// rule that needs to reason about "now". Adding an unused clock.Clock
// parameter here just because other constructors have one would be
// exactly the unnecessary dependency CLAUDE.md warns against.
type SiteService struct {
	sites inventory.SiteRepository
}

// NewSiteService builds a SiteService.
func NewSiteService(sites inventory.SiteRepository) *SiteService {
	return &SiteService{sites: sites}
}

// Get retrieves a Site by ID.
func (s *SiteService) Get(ctx context.Context, id uuid.UUID) (inventory.Site, error) {
	return s.sites.Get(ctx, id)
}

// List returns every Site.
func (s *SiteService) List(ctx context.Context) ([]inventory.Site, error) {
	return s.sites.List(ctx)
}

// Create validates site and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of SiteService (e.g. a workflow step
// that creates a Site programmatically, with no HTTP request involved at
// all) — so every caller gets the same guarantee for free, and invalid
// input never costs a database round trip.
func (s *SiteService) Create(ctx context.Context, site inventory.Site) (inventory.Site, error) {
	if err := site.Validate(); err != nil {
		return inventory.Site{}, err
	}
	return s.sites.Create(ctx, site)
}

// Update validates site and, if valid, persists the change. See Create
// for why validation happens here rather than elsewhere.
func (s *SiteService) Update(ctx context.Context, site inventory.Site) (inventory.Site, error) {
	if err := site.Validate(); err != nil {
		return inventory.Site{}, err
	}
	return s.sites.Update(ctx, site)
}

// Delete removes the Site identified by id.
func (s *SiteService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.sites.Delete(ctx, id)
}
