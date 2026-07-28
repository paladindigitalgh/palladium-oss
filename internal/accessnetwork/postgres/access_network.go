// Package postgres implements the Access Network domain's
// AccessNetworkRepository against PostgreSQL using pgx directly — no ORM
// — following the exact pattern established by
// internal/catalog/postgres.CatalogRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// AccessNetworkRepository implements
// accessnetwork.AccessNetworkRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type AccessNetworkRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ accessnetwork.AccessNetworkRepository = (*AccessNetworkRepository)(nil)

// NewAccessNetworkRepository builds an AccessNetworkRepository.
func NewAccessNetworkRepository(db database.Querier, clock clock.Clock, ids id.Generator) *AccessNetworkRepository {
	return &AccessNetworkRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves an AccessNetwork by ID, or an apperror.KindNotFound
// error if none exists.
func (r *AccessNetworkRepository) Get(ctx context.Context, networkID uuid.UUID) (accessnetwork.AccessNetwork, error) {
	const query = `
		SELECT id, name, status, description, created_at, updated_at
		FROM access_networks
		WHERE id = $1
	`

	a, err := scanAccessNetwork(r.db.QueryRow(ctx, query, networkID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessnetwork.AccessNetwork{}, accessNetworkNotFound(networkID)
		}
		return accessnetwork.AccessNetwork{}, translateError("get access network", err)
	}
	return a, nil
}

// List returns every AccessNetwork, ordered by name for stable,
// human-useful output (see the index added on that column in the
// migration).
func (r *AccessNetworkRepository) List(ctx context.Context) ([]accessnetwork.AccessNetwork, error) {
	const query = `
		SELECT id, name, status, description, created_at, updated_at
		FROM access_networks
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list access networks", err)
	}
	defer rows.Close()

	networks := []accessnetwork.AccessNetwork{}
	for rows.Next() {
		a, err := scanAccessNetwork(rows)
		if err != nil {
			return nil, translateError("scan access network row", err)
		}
		networks = append(networks, a)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list access networks", err)
	}

	return networks, nil
}

// Create inserts a and returns the persisted record.
//
// As with CatalogRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input
// AccessNetwork for those fields are ignored. Status is taken from the
// input exactly as given; the repository has no business logic and does
// not decide it.
func (r *AccessNetworkRepository) Create(ctx context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	const query = `
		INSERT INTO access_networks (id, name, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, name, status, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanAccessNetwork(r.db.QueryRow(ctx, query,
		r.ids.New(), a.Name, string(a.Status), a.Description, now))
	if err != nil {
		return accessnetwork.AccessNetwork{}, translateError("create access network", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (Name, Status, Description) of the
// AccessNetwork identified by a.ID and returns the persisted record, or
// an apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input AccessNetwork contained.
func (r *AccessNetworkRepository) Update(ctx context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	const query = `
		UPDATE access_networks
		SET name = $1, status = $2, description = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, name, status, description, created_at, updated_at
	`

	updated, err := scanAccessNetwork(r.db.QueryRow(ctx, query,
		a.Name, string(a.Status), a.Description, r.clock.Now(), a.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessnetwork.AccessNetwork{}, accessNetworkNotFound(a.ID)
		}
		return accessnetwork.AccessNetwork{}, translateError("update access network", err)
	}
	return updated, nil
}

// Delete removes the AccessNetwork identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If any OLT still
// references this AccessNetwork, the foreign key's ON DELETE RESTRICT
// rejects the delete and this returns an apperror.KindConflict error
// instead.
func (r *AccessNetworkRepository) Delete(ctx context.Context, networkID uuid.UUID) error {
	const query = `DELETE FROM access_networks WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, networkID)
	if err != nil {
		return translateError("delete access network", err)
	}
	if tag.RowsAffected() == 0 {
		return accessNetworkNotFound(networkID)
	}
	return nil
}

func accessNetworkNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("access network %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanAccessNetwork backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccessNetwork(row rowScanner) (accessnetwork.AccessNetwork, error) {
	var (
		a      accessnetwork.AccessNetwork
		status string
	)
	err := row.Scan(&a.ID, &a.Name, &status, &a.Description, &a.CreatedAt, &a.UpdatedAt)
	a.Status = accessnetwork.AccessNetworkStatus(status)
	return a, err
}
