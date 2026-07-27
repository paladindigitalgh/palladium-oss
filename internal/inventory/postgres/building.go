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

// BuildingRepository implements inventory.BuildingRepository against
// PostgreSQL. It follows SiteRepository (site.go) exactly; see that file
// for the reasoning behind depending on database.Querier and injecting
// clock/ids, which is not repeated here.
type BuildingRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ inventory.BuildingRepository = (*BuildingRepository)(nil)

// NewBuildingRepository builds a BuildingRepository.
func NewBuildingRepository(db database.Querier, clock clock.Clock, ids id.Generator) *BuildingRepository {
	return &BuildingRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Building by ID, or an apperror.KindNotFound error if none
// exists.
func (r *BuildingRepository) Get(ctx context.Context, buildingID uuid.UUID) (inventory.Building, error) {
	const query = `
		SELECT id, site_id, name, description, created_at, updated_at
		FROM buildings
		WHERE id = $1
	`

	building, err := scanBuilding(r.db.QueryRow(ctx, query, buildingID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Building{}, buildingNotFound(buildingID)
		}
		return inventory.Building{}, translateError("get building", err)
	}
	return building, nil
}

// List returns every Building, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *BuildingRepository) List(ctx context.Context) ([]inventory.Building, error) {
	const query = `
		SELECT id, site_id, name, description, created_at, updated_at
		FROM buildings
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list buildings", err)
	}
	defer rows.Close()

	buildings := []inventory.Building{}
	for rows.Next() {
		building, err := scanBuilding(rows)
		if err != nil {
			return nil, translateError("scan building row", err)
		}
		buildings = append(buildings, building)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list buildings", err)
	}

	return buildings, nil
}

// Create inserts building and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt, and
// UpdatedAt itself; any values already set on the input Building for those
// fields are ignored. A SiteID that does not reference an existing Site
// fails with an apperror.KindConflict error (see translateError).
func (r *BuildingRepository) Create(ctx context.Context, building inventory.Building) (inventory.Building, error) {
	const query = `
		INSERT INTO buildings (id, site_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, site_id, name, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanBuilding(r.db.QueryRow(ctx, query,
		r.ids.New(), building.SiteID, building.Name, building.Description, now))
	if err != nil {
		return inventory.Building{}, translateError("create building", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (SiteID, Name, Description) of the
// Building identified by building.ID and returns the persisted record, or
// an apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method, for the same reason as
// SiteRepository.Update. SiteID is treated as mutable, consistent with how
// Site treats every field but ID and CreatedAt as caller-controlled: the
// repository has no business logic and does not decide that reassigning a
// Building to a different Site should be forbidden.
func (r *BuildingRepository) Update(ctx context.Context, building inventory.Building) (inventory.Building, error) {
	const query = `
		UPDATE buildings
		SET site_id = $1, name = $2, description = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, site_id, name, description, created_at, updated_at
	`

	updated, err := scanBuilding(r.db.QueryRow(ctx, query,
		building.SiteID, building.Name, building.Description, r.clock.Now(), building.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Building{}, buildingNotFound(building.ID)
		}
		return inventory.Building{}, translateError("update building", err)
	}
	return updated, nil
}

// Delete removes the Building identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any Room still
// references this Building, the foreign key's ON DELETE RESTRICT rejects
// the delete and this returns an apperror.KindConflict error instead.
func (r *BuildingRepository) Delete(ctx context.Context, buildingID uuid.UUID) error {
	const query = `DELETE FROM buildings WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, buildingID)
	if err != nil {
		return translateError("delete building", err)
	}
	if tag.RowsAffected() == 0 {
		return buildingNotFound(buildingID)
	}
	return nil
}

func buildingNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("building %s not found", id))
}

func scanBuilding(row rowScanner) (inventory.Building, error) {
	var building inventory.Building
	err := row.Scan(&building.ID, &building.SiteID, &building.Name, &building.Description,
		&building.CreatedAt, &building.UpdatedAt)
	return building, err
}
