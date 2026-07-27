// Package postgres implements Inventory domain repositories against
// PostgreSQL using pgx directly — no ORM (see CLAUDE.md's Database Rules
// and this milestone's requirements). Only SiteRepository exists so far;
// Building, Room, Rack, and Device persistence are deliberately out of
// scope here and will follow the same pattern established by this file.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// SiteRepository implements inventory.SiteRepository against PostgreSQL.
type SiteRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ inventory.SiteRepository = (*SiteRepository)(nil)

// NewSiteRepository builds a SiteRepository.
//
// db is database.Querier, not *database.Pool: a repository that only
// depends on the narrower interface can be constructed either directly on
// the pool (normal use) or on a pgx.Tx obtained from database.RunInTx (see
// internal/database/querier.go), so a future service layer can include Site
// writes in the same transaction as other repositories' writes without this
// package knowing anything about transactions.
//
// clock and ids are injected rather than called directly (time.Now,
// uuid.New) so timestamps and identity generation stay swappable and
// testable — see internal/platform/clock and internal/platform/id, which
// exist specifically for this.
func NewSiteRepository(db database.Querier, clock clock.Clock, ids id.Generator) *SiteRepository {
	return &SiteRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Site by ID, or an apperror.KindNotFound error if none
// exists.
func (r *SiteRepository) Get(ctx context.Context, siteID uuid.UUID) (inventory.Site, error) {
	const query = `
		SELECT id, name, description, created_at, updated_at
		FROM sites
		WHERE id = $1
	`

	site, err := scanSite(r.db.QueryRow(ctx, query, siteID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Site{}, siteNotFound(siteID)
		}
		return inventory.Site{}, translateError("get site", err)
	}
	return site, nil
}

// List returns every Site, ordered by name for stable, human-useful output
// (see the index added on that column in the migration).
func (r *SiteRepository) List(ctx context.Context) ([]inventory.Site, error) {
	const query = `
		SELECT id, name, description, created_at, updated_at
		FROM sites
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list sites", err)
	}
	defer rows.Close()

	sites := []inventory.Site{}
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, translateError("scan site row", err)
		}
		sites = append(sites, site)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list sites", err)
	}

	return sites, nil
}

// Create inserts site and returns the persisted record.
//
// The repository assigns ID, CreatedAt, and UpdatedAt itself — any values
// already set on the input Site for those fields are ignored. Identity and
// creation time are metadata this layer owns, not something a caller can be
// trusted to supply correctly (a caller-supplied ID could collide or be
// predictable; a caller-supplied CreatedAt could misrepresent history,
// which matters given CLAUDE.md's audit expectations).
func (r *SiteRepository) Create(ctx context.Context, site inventory.Site) (inventory.Site, error) {
	const query = `
		INSERT INTO sites (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id, name, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanSite(r.db.QueryRow(ctx, query, r.ids.New(), site.Name, site.Description, now))
	if err != nil {
		return inventory.Site{}, translateError("create site", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (Name, Description) of the Site
// identified by site.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Site contained.
func (r *SiteRepository) Update(ctx context.Context, site inventory.Site) (inventory.Site, error) {
	const query = `
		UPDATE sites
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, name, description, created_at, updated_at
	`

	updated, err := scanSite(r.db.QueryRow(ctx, query, site.Name, site.Description, r.clock.Now(), site.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Site{}, siteNotFound(site.ID)
		}
		return inventory.Site{}, translateError("update site", err)
	}
	return updated, nil
}

// Delete removes the Site identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *SiteRepository) Delete(ctx context.Context, siteID uuid.UUID) error {
	const query = `DELETE FROM sites WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, siteID)
	if err != nil {
		return translateError("delete site", err)
	}
	if tag.RowsAffected() == 0 {
		return siteNotFound(siteID)
	}
	return nil
}

func siteNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("site %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanSite backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanSite(row rowScanner) (inventory.Site, error) {
	var site inventory.Site
	err := row.Scan(&site.ID, &site.Name, &site.Description, &site.CreatedAt, &site.UpdatedAt)
	return site, err
}
