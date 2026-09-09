// Package postgres implements the Device Manufacturer domain's
// DeviceManufacturerRepository against PostgreSQL using pgx directly —
// no ORM — following the exact pattern established by
// internal/oltmodel/postgres.OLTModelRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// DeviceManufacturerRepository implements
// devicemanufacturer.DeviceManufacturerRepository against PostgreSQL.
// See internal/inventory/postgres/site.go for the reasoning behind
// depending on database.Querier and injecting clock/ids, which is not
// repeated here.
type DeviceManufacturerRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ devicemanufacturer.DeviceManufacturerRepository = (*DeviceManufacturerRepository)(nil)

// NewDeviceManufacturerRepository builds a DeviceManufacturerRepository.
func NewDeviceManufacturerRepository(db database.Querier, clock clock.Clock, ids id.Generator) *DeviceManufacturerRepository {
	return &DeviceManufacturerRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a DeviceManufacturer by ID, or an apperror.KindNotFound
// error if none exists.
func (r *DeviceManufacturerRepository) Get(ctx context.Context, manufacturerID uuid.UUID) (devicemanufacturer.DeviceManufacturer, error) {
	const query = `
		SELECT id, name, description, created_at, updated_at
		FROM device_manufacturers
		WHERE id = $1
	`

	m, err := scanDeviceManufacturer(r.db.QueryRow(ctx, query, manufacturerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return devicemanufacturer.DeviceManufacturer{}, deviceManufacturerNotFound(manufacturerID)
		}
		return devicemanufacturer.DeviceManufacturer{}, translateError("get device manufacturer", err)
	}
	return m, nil
}

// List returns every DeviceManufacturer, ordered by name for stable,
// human-useful output (see the index added on that column in the
// migration).
func (r *DeviceManufacturerRepository) List(ctx context.Context) ([]devicemanufacturer.DeviceManufacturer, error) {
	const query = `
		SELECT id, name, description, created_at, updated_at
		FROM device_manufacturers
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list device manufacturers", err)
	}
	defer rows.Close()

	manufacturers := []devicemanufacturer.DeviceManufacturer{}
	for rows.Next() {
		m, err := scanDeviceManufacturer(rows)
		if err != nil {
			return nil, translateError("scan device manufacturer row", err)
		}
		manufacturers = append(manufacturers, m)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list device manufacturers", err)
	}

	return manufacturers, nil
}

// Create inserts m and returns the persisted record.
//
// The repository assigns ID, CreatedAt, and UpdatedAt itself — any
// values already set on the input DeviceManufacturer for those fields
// are ignored.
func (r *DeviceManufacturerRepository) Create(ctx context.Context, m devicemanufacturer.DeviceManufacturer) (devicemanufacturer.DeviceManufacturer, error) {
	const query = `
		INSERT INTO device_manufacturers (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id, name, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanDeviceManufacturer(r.db.QueryRow(ctx, query, r.ids.New(), m.Name, m.Description, now))
	if err != nil {
		return devicemanufacturer.DeviceManufacturer{}, translateError("create device manufacturer", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the DeviceManufacturer
// identified by m.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input DeviceManufacturer
// contained.
func (r *DeviceManufacturerRepository) Update(ctx context.Context, m devicemanufacturer.DeviceManufacturer) (devicemanufacturer.DeviceManufacturer, error) {
	const query = `
		UPDATE device_manufacturers
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, name, description, created_at, updated_at
	`

	updated, err := scanDeviceManufacturer(r.db.QueryRow(ctx, query, m.Name, m.Description, r.clock.Now(), m.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return devicemanufacturer.DeviceManufacturer{}, deviceManufacturerNotFound(m.ID)
		}
		return devicemanufacturer.DeviceManufacturer{}, translateError("update device manufacturer", err)
	}
	return updated, nil
}

// Delete removes the DeviceManufacturer identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any DeviceModel
// still references this DeviceManufacturer, the foreign key's ON DELETE
// RESTRICT rejects the delete and this returns an apperror.KindConflict
// error instead.
func (r *DeviceManufacturerRepository) Delete(ctx context.Context, manufacturerID uuid.UUID) error {
	const query = `DELETE FROM device_manufacturers WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, manufacturerID)
	if err != nil {
		return translateError("delete device manufacturer", err)
	}
	if tag.RowsAffected() == 0 {
		return deviceManufacturerNotFound(manufacturerID)
	}
	return nil
}

func deviceManufacturerNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("device manufacturer %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanDeviceManufacturer backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanDeviceManufacturer(row rowScanner) (devicemanufacturer.DeviceManufacturer, error) {
	var m devicemanufacturer.DeviceManufacturer
	err := row.Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}
