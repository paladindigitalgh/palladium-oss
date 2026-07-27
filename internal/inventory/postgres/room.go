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

// RoomRepository implements inventory.RoomRepository against PostgreSQL. It
// follows SiteRepository (site.go) exactly; see that file for the
// reasoning behind depending on database.Querier and injecting clock/ids,
// which is not repeated here.
type RoomRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ inventory.RoomRepository = (*RoomRepository)(nil)

// NewRoomRepository builds a RoomRepository.
func NewRoomRepository(db database.Querier, clock clock.Clock, ids id.Generator) *RoomRepository {
	return &RoomRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Room by ID, or an apperror.KindNotFound error if none
// exists.
func (r *RoomRepository) Get(ctx context.Context, roomID uuid.UUID) (inventory.Room, error) {
	const query = `
		SELECT id, building_id, name, description, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`

	room, err := scanRoom(r.db.QueryRow(ctx, query, roomID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Room{}, roomNotFound(roomID)
		}
		return inventory.Room{}, translateError("get room", err)
	}
	return room, nil
}

// List returns every Room, ordered by name for stable, human-useful output
// (see the index added on that column in the migration).
func (r *RoomRepository) List(ctx context.Context) ([]inventory.Room, error) {
	const query = `
		SELECT id, building_id, name, description, created_at, updated_at
		FROM rooms
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list rooms", err)
	}
	defer rows.Close()

	rooms := []inventory.Room{}
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, translateError("scan room row", err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list rooms", err)
	}

	return rooms, nil
}

// Create inserts room and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt, and
// UpdatedAt itself; any values already set on the input Room for those
// fields are ignored. A BuildingID that does not reference an existing
// Building fails with an apperror.KindConflict error (see translateError).
func (r *RoomRepository) Create(ctx context.Context, room inventory.Room) (inventory.Room, error) {
	const query = `
		INSERT INTO rooms (id, building_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, building_id, name, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanRoom(r.db.QueryRow(ctx, query,
		r.ids.New(), room.BuildingID, room.Name, room.Description, now))
	if err != nil {
		return inventory.Room{}, translateError("create room", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (BuildingID, Name, Description) of
// the Room identified by room.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method, for the same reason as
// SiteRepository.Update. BuildingID is treated as mutable, for the same
// reason BuildingRepository.Update treats SiteID as mutable.
func (r *RoomRepository) Update(ctx context.Context, room inventory.Room) (inventory.Room, error) {
	const query = `
		UPDATE rooms
		SET building_id = $1, name = $2, description = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, building_id, name, description, created_at, updated_at
	`

	updated, err := scanRoom(r.db.QueryRow(ctx, query,
		room.BuildingID, room.Name, room.Description, r.clock.Now(), room.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Room{}, roomNotFound(room.ID)
		}
		return inventory.Room{}, translateError("update room", err)
	}
	return updated, nil
}

// Delete removes the Room identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any Rack still
// references this Room, the foreign key's ON DELETE RESTRICT rejects the
// delete and this returns an apperror.KindConflict error instead.
func (r *RoomRepository) Delete(ctx context.Context, roomID uuid.UUID) error {
	const query = `DELETE FROM rooms WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, roomID)
	if err != nil {
		return translateError("delete room", err)
	}
	if tag.RowsAffected() == 0 {
		return roomNotFound(roomID)
	}
	return nil
}

func roomNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("room %s not found", id))
}

func scanRoom(row rowScanner) (inventory.Room, error) {
	var room inventory.Room
	err := row.Scan(&room.ID, &room.BuildingID, &room.Name, &room.Description,
		&room.CreatedAt, &room.UpdatedAt)
	return room, err
}
