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

// RackRepository implements inventory.RackRepository against PostgreSQL. It
// follows SiteRepository (site.go) exactly; see that file for the
// reasoning behind depending on database.Querier and injecting clock/ids,
// which is not repeated here.
type RackRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ inventory.RackRepository = (*RackRepository)(nil)

// NewRackRepository builds a RackRepository.
func NewRackRepository(db database.Querier, clock clock.Clock, ids id.Generator) *RackRepository {
	return &RackRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Rack by ID, or an apperror.KindNotFound error if none
// exists.
func (r *RackRepository) Get(ctx context.Context, rackID uuid.UUID) (inventory.Rack, error) {
	const query = `
		SELECT id, room_id, name, description, created_at, updated_at
		FROM racks
		WHERE id = $1
	`

	rack, err := scanRack(r.db.QueryRow(ctx, query, rackID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Rack{}, rackNotFound(rackID)
		}
		return inventory.Rack{}, translateError("get rack", err)
	}
	return rack, nil
}

// List returns every Rack, ordered by name for stable, human-useful output
// (see the index added on that column in the migration).
func (r *RackRepository) List(ctx context.Context) ([]inventory.Rack, error) {
	const query = `
		SELECT id, room_id, name, description, created_at, updated_at
		FROM racks
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list racks", err)
	}
	defer rows.Close()

	racks := []inventory.Rack{}
	for rows.Next() {
		rack, err := scanRack(rows)
		if err != nil {
			return nil, translateError("scan rack row", err)
		}
		racks = append(racks, rack)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list racks", err)
	}

	return racks, nil
}

// Create inserts rack and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt, and
// UpdatedAt itself; any values already set on the input Rack for those
// fields are ignored. RoomID may be nil (see inventory.Rack); a non-nil
// RoomID that does not reference an existing Room fails with an
// apperror.KindConflict error (see translateError).
func (r *RackRepository) Create(ctx context.Context, rack inventory.Rack) (inventory.Rack, error) {
	const query = `
		INSERT INTO racks (id, room_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, room_id, name, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanRack(r.db.QueryRow(ctx, query,
		r.ids.New(), rack.RoomID, rack.Name, rack.Description, now))
	if err != nil {
		return inventory.Rack{}, translateError("create rack", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (RoomID, Name, Description) of the
// Rack identified by rack.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method, for the same reason as
// SiteRepository.Update. RoomID is treated as mutable, for the same reason
// BuildingRepository.Update treats SiteID as mutable — this is also how a
// Rack transitions from unracked (nil RoomID) to installed.
func (r *RackRepository) Update(ctx context.Context, rack inventory.Rack) (inventory.Rack, error) {
	const query = `
		UPDATE racks
		SET room_id = $1, name = $2, description = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, room_id, name, description, created_at, updated_at
	`

	updated, err := scanRack(r.db.QueryRow(ctx, query,
		rack.RoomID, rack.Name, rack.Description, r.clock.Now(), rack.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Rack{}, rackNotFound(rack.ID)
		}
		return inventory.Rack{}, translateError("update rack", err)
	}
	return updated, nil
}

// Delete removes the Rack identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any Device still
// references this Rack, the foreign key's ON DELETE RESTRICT rejects the
// delete and this returns an apperror.KindConflict error instead.
func (r *RackRepository) Delete(ctx context.Context, rackID uuid.UUID) error {
	const query = `DELETE FROM racks WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, rackID)
	if err != nil {
		return translateError("delete rack", err)
	}
	if tag.RowsAffected() == 0 {
		return rackNotFound(rackID)
	}
	return nil
}

func rackNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("rack %s not found", id))
}

func scanRack(row rowScanner) (inventory.Rack, error) {
	var rack inventory.Rack
	err := row.Scan(&rack.ID, &rack.RoomID, &rack.Name, &rack.Description,
		&rack.CreatedAt, &rack.UpdatedAt)
	return rack, err
}
