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
		SELECT id, manufacturer_id, name, description, is_default, created_at, updated_at
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
		SELECT id, manufacturer_id, name, description, is_default, created_at, updated_at
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
// translateError). IsDefault is likewise ignored and always inserted
// false: becoming the default is exclusively SetDefault's job (see that
// method's own doc comment on why), never a side effect of creating a
// new entry.
func (r *DeviceModelRepository) Create(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error) {
	const query = `
		INSERT INTO device_models (id, manufacturer_id, name, description, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, false, $5, $5)
		RETURNING id, manufacturer_id, name, description, is_default, created_at, updated_at
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
// is_default is the same story, deliberately: this statement never
// assigns that column either, so an ordinary edit can never accidentally
// clear (or silently set) the current default — only SetDefault touches
// is_default. m.IsDefault is therefore ignored entirely; the RETURNING
// clause still reports the real stored value.
func (r *DeviceModelRepository) Update(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error) {
	const query = `
		UPDATE device_models
		SET manufacturer_id = $1, name = $2, description = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, manufacturer_id, name, description, is_default, created_at, updated_at
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

// SetDefault sets or clears is_default on the DeviceModel identified by
// id, or returns an apperror.KindNotFound error if it does not exist.
//
// Setting isDefault true is a single unconditional UPDATE scoped to
// every DeviceModel sharing id's own manufacturer_id (a subquery against
// id itself, not a caller-supplied value -- so this can never be pointed
// at the wrong Manufacturer's rows by mistake): is_default = (id = $1)
// evaluates per row from each row's own pre-statement state, so this
// both sets id's own is_default true and clears every sibling's in one
// atomic statement, scoped to that one Manufacturer only (see
// DeviceModel.IsDefault's own doc comment on why per-manufacturer, not
// system-wide). Setting isDefault false only ever touches id.
func (r *DeviceModelRepository) SetDefault(ctx context.Context, id uuid.UUID, isDefault bool) error {
	if isDefault {
		const query = `
			UPDATE device_models
			SET is_default = (id = $1), updated_at = now()
			WHERE manufacturer_id = (SELECT manufacturer_id FROM device_models WHERE id = $1)
		`
		if _, err := r.db.Exec(ctx, query, id); err != nil {
			return translateError("set device model default", err)
		}
		// The statement above always "succeeds" even for an id matching no
		// row at all -- the subquery just returns no manufacturer_id, so
		// the WHERE clause matches nothing and RowsAffected is 0 either
		// way, indistinguishable from "id exists but has no siblings."
		// Confirm existence explicitly rather than trust that count.
		if _, err := r.Get(ctx, id); err != nil {
			return err
		}
		return nil
	}

	const query = `UPDATE device_models SET is_default = false, updated_at = now() WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return translateError("set device model default", err)
	}
	if tag.RowsAffected() == 0 {
		return deviceModelNotFound(id)
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
	err := row.Scan(&m.ID, &m.ManufacturerID, &m.Name, &m.Description, &m.IsDefault, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}
