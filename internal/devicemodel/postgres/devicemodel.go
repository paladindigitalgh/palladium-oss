// Package postgres implements the Device Model domain's
// DeviceModelRepository against PostgreSQL using pgx directly — no
// ORM — following the exact pattern established by
// internal/oltmodel/postgres.OLTModelRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/devicemodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// DeviceModelRepository implements devicemodel.DeviceModelRepository
// against PostgreSQL. See internal/inventory/postgres/site.go for the
// reasoning behind depending on database.Querier and injecting
// clock/ids, which is not repeated here.
type DeviceModelRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ devicemodel.DeviceModelRepository = (*DeviceModelRepository)(nil)

// NewDeviceModelRepository builds a DeviceModelRepository.
func NewDeviceModelRepository(db database.Querier, clock clock.Clock, ids id.Generator) *DeviceModelRepository {
	return &DeviceModelRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a DeviceModel by ID, or an apperror.KindNotFound error
// if none exists.
func (r *DeviceModelRepository) Get(ctx context.Context, modelID uuid.UUID) (devicemodel.DeviceModel, error) {
	const query = `
		SELECT id, manufacturer_id, name, description, created_at, updated_at
		FROM device_models
		WHERE id = $1
	`

	m, err := scanDeviceModel(r.db.QueryRow(ctx, query, modelID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return devicemodel.DeviceModel{}, deviceModelNotFound(modelID)
		}
		return devicemodel.DeviceModel{}, translateError("get device model", err)
	}
	return m, nil
}

// List returns every DeviceModel, ordered by name for stable,
// human-useful output (see the index added on that column in the
// migration).
func (r *DeviceModelRepository) List(ctx context.Context) ([]devicemodel.DeviceModel, error) {
	const query = `
		SELECT id, manufacturer_id, name, description, created_at, updated_at
		FROM device_models
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list device models", err)
	}
	defer rows.Close()

	models := []devicemodel.DeviceModel{}
	for rows.Next() {
		m, err := scanDeviceModel(rows)
		if err != nil {
			return nil, translateError("scan device model row", err)
		}
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list device models", err)
	}

	return models, nil
}

// Create inserts m and returns the persisted record.
//
// The repository assigns ID, CreatedAt, and UpdatedAt itself — any
// values already set on the input DeviceModel for those fields are
// ignored. A ManufacturerID that does not reference an existing
// DeviceManufacturer fails with an apperror.KindConflict error (see
// translateError).
func (r *DeviceModelRepository) Create(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error) {
	const query = `
		INSERT INTO device_models (id, manufacturer_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, manufacturer_id, name, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanDeviceModel(r.db.QueryRow(ctx, query, r.ids.New(), m.ManufacturerID, m.Name, m.Description, now))
	if err != nil {
		return devicemodel.DeviceModel{}, translateError("create device model", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the DeviceModel identified by
// m.ID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input DeviceModel contained.
func (r *DeviceModelRepository) Update(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error) {
	const query = `
		UPDATE device_models
		SET manufacturer_id = $1, name = $2, description = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, manufacturer_id, name, description, created_at, updated_at
	`

	updated, err := scanDeviceModel(r.db.QueryRow(ctx, query, m.ManufacturerID, m.Name, m.Description, r.clock.Now(), m.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return devicemodel.DeviceModel{}, deviceModelNotFound(m.ID)
		}
		return devicemodel.DeviceModel{}, translateError("update device model", err)
	}
	return updated, nil
}

// Delete removes the DeviceModel identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any Device still
// references this DeviceModel, the foreign key's ON DELETE RESTRICT
// rejects the delete and this returns an apperror.KindConflict error
// instead.
func (r *DeviceModelRepository) Delete(ctx context.Context, modelID uuid.UUID) error {
	const query = `DELETE FROM device_models WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, modelID)
	if err != nil {
		return translateError("delete device model", err)
	}
	if tag.RowsAffected() == 0 {
		return deviceModelNotFound(modelID)
	}
	return nil
}

func deviceModelNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("device model %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanDeviceModel backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanDeviceModel(row rowScanner) (devicemodel.DeviceModel, error) {
	var m devicemodel.DeviceModel
	err := row.Scan(&m.ID, &m.ManufacturerID, &m.Name, &m.Description, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}
