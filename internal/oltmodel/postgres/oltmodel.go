// Package postgres implements the OLT Model domain's OLTModelRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/provider/postgres.ProviderRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// OLTModelRepository implements oltmodel.OLTModelRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type OLTModelRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ oltmodel.OLTModelRepository = (*OLTModelRepository)(nil)

// NewOLTModelRepository builds an OLTModelRepository.
func NewOLTModelRepository(db database.Querier, clock clock.Clock, ids id.Generator) *OLTModelRepository {
	return &OLTModelRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves an OLTModel by ID, or an apperror.KindNotFound error if
// none exists.
func (r *OLTModelRepository) Get(ctx context.Context, modelID uuid.UUID) (oltmodel.OLTModel, error) {
	const query = `
		SELECT id, vendor, name, pon_port_count, description, created_at, updated_at
		FROM olt_models
		WHERE id = $1
	`

	m, err := scanOLTModel(r.db.QueryRow(ctx, query, modelID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return oltmodel.OLTModel{}, oltModelNotFound(modelID)
		}
		return oltmodel.OLTModel{}, translateError("get olt model", err)
	}
	return m, nil
}

// List returns every OLTModel, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *OLTModelRepository) List(ctx context.Context) ([]oltmodel.OLTModel, error) {
	const query = `
		SELECT id, vendor, name, pon_port_count, description, created_at, updated_at
		FROM olt_models
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list olt models", err)
	}
	defer rows.Close()

	models := []oltmodel.OLTModel{}
	for rows.Next() {
		m, err := scanOLTModel(rows)
		if err != nil {
			return nil, translateError("scan olt model row", err)
		}
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list olt models", err)
	}

	return models, nil
}

// Create inserts m and returns the persisted record.
//
// The repository assigns ID, CreatedAt, and UpdatedAt itself — any
// values already set on the input OLTModel for those fields are
// ignored.
func (r *OLTModelRepository) Create(ctx context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	const query = `
		INSERT INTO olt_models (id, vendor, name, pon_port_count, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, vendor, name, pon_port_count, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanOLTModel(r.db.QueryRow(ctx, query,
		r.ids.New(), string(m.Vendor), m.Name, m.PONPortCount, m.Description, now))
	if err != nil {
		return oltmodel.OLTModel{}, translateError("create olt model", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the OLTModel identified by
// m.ID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input OLTModel contained.
func (r *OLTModelRepository) Update(ctx context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	const query = `
		UPDATE olt_models
		SET vendor = $1, name = $2, pon_port_count = $3, description = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, vendor, name, pon_port_count, description, created_at, updated_at
	`

	updated, err := scanOLTModel(r.db.QueryRow(ctx, query,
		string(m.Vendor), m.Name, m.PONPortCount, m.Description, r.clock.Now(), m.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return oltmodel.OLTModel{}, oltModelNotFound(m.ID)
		}
		return oltmodel.OLTModel{}, translateError("update olt model", err)
	}
	return updated, nil
}

// Delete removes the OLTModel identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any OLT still
// references this OLTModel, the foreign key's ON DELETE RESTRICT
// rejects the delete and this returns an apperror.KindConflict error
// instead.
func (r *OLTModelRepository) Delete(ctx context.Context, modelID uuid.UUID) error {
	const query = `DELETE FROM olt_models WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, modelID)
	if err != nil {
		return translateError("delete olt model", err)
	}
	if tag.RowsAffected() == 0 {
		return oltModelNotFound(modelID)
	}
	return nil
}

func oltModelNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("olt model %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanOLTModel backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanOLTModel(row rowScanner) (oltmodel.OLTModel, error) {
	var (
		m      oltmodel.OLTModel
		vendor string
	)
	err := row.Scan(&m.ID, &vendor, &m.Name, &m.PONPortCount, &m.Description, &m.CreatedAt, &m.UpdatedAt)
	m.Vendor = oltmodel.Vendor(vendor)
	return m, err
}
