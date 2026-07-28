// Package postgres implements the OLT domain's OLTRepository against
// PostgreSQL using pgx directly — no ORM — following the exact pattern
// established by internal/product/postgres.ProductRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// OLTRepository implements olt.OLTRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type OLTRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ olt.OLTRepository = (*OLTRepository)(nil)

// NewOLTRepository builds an OLTRepository.
func NewOLTRepository(db database.Querier, clock clock.Clock, ids id.Generator) *OLTRepository {
	return &OLTRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves an OLT by ID, or an apperror.KindNotFound error if none
// exists.
func (r *OLTRepository) Get(ctx context.Context, oltID uuid.UUID) (olt.OLT, error) {
	const query = `
		SELECT id, access_network_id, name, vendor, model, management_ip_address,
		       description, created_at, updated_at
		FROM olts
		WHERE id = $1
	`

	o, err := scanOLT(r.db.QueryRow(ctx, query, oltID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return olt.OLT{}, oltNotFound(oltID)
		}
		return olt.OLT{}, translateError("get olt", err)
	}
	return o, nil
}

// List returns every OLT, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *OLTRepository) List(ctx context.Context) ([]olt.OLT, error) {
	const query = `
		SELECT id, access_network_id, name, vendor, model, management_ip_address,
		       description, created_at, updated_at
		FROM olts
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list olts", err)
	}
	defer rows.Close()

	olts := []olt.OLT{}
	for rows.Next() {
		o, err := scanOLT(rows)
		if err != nil {
			return nil, translateError("scan olt row", err)
		}
		olts = append(olts, o)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list olts", err)
	}

	return olts, nil
}

// Create inserts o and returns the persisted record.
//
// As with ProductRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input OLT for
// those fields are ignored. An AccessNetworkID that does not reference
// an existing AccessNetwork fails with an apperror.KindConflict error
// (see translateError).
func (r *OLTRepository) Create(ctx context.Context, o olt.OLT) (olt.OLT, error) {
	const query = `
		INSERT INTO olts (
			id, access_network_id, name, vendor, model, management_ip_address,
			description, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING id, access_network_id, name, vendor, model, management_ip_address,
		          description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanOLT(r.db.QueryRow(ctx, query,
		r.ids.New(), o.AccessNetworkID, o.Name, string(o.Vendor), o.Model, o.ManagementIPAddress,
		o.Description, now))
	if err != nil {
		return olt.OLT{}, translateError("create olt", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the OLT identified by o.ID and
// returns the persisted record, or an apperror.KindNotFound error if it
// does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input OLT contained.
func (r *OLTRepository) Update(ctx context.Context, o olt.OLT) (olt.OLT, error) {
	const query = `
		UPDATE olts
		SET access_network_id = $1, name = $2, vendor = $3, model = $4, management_ip_address = $5,
		    description = $6, updated_at = $7
		WHERE id = $8
		RETURNING id, access_network_id, name, vendor, model, management_ip_address,
		          description, created_at, updated_at
	`

	updated, err := scanOLT(r.db.QueryRow(ctx, query,
		o.AccessNetworkID, o.Name, string(o.Vendor), o.Model, o.ManagementIPAddress,
		o.Description, r.clock.Now(), o.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return olt.OLT{}, oltNotFound(o.ID)
		}
		return olt.OLT{}, translateError("update olt", err)
	}
	return updated, nil
}

// Delete removes the OLT identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any PONPort still
// references this OLT, the foreign key's ON DELETE RESTRICT rejects the
// delete and this returns an apperror.KindConflict error instead.
func (r *OLTRepository) Delete(ctx context.Context, oltID uuid.UUID) error {
	const query = `DELETE FROM olts WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, oltID)
	if err != nil {
		return translateError("delete olt", err)
	}
	if tag.RowsAffected() == 0 {
		return oltNotFound(oltID)
	}
	return nil
}

func oltNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("olt %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanOLT backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanOLT(row rowScanner) (olt.OLT, error) {
	var (
		o      olt.OLT
		vendor string
	)
	err := row.Scan(
		&o.ID, &o.AccessNetworkID, &o.Name, &vendor, &o.Model, &o.ManagementIPAddress,
		&o.Description, &o.CreatedAt, &o.UpdatedAt,
	)
	o.Vendor = olt.Vendor(vendor)
	return o, err
}
