// Package postgres implements the Access Interface domain's
// AccessInterfaceRepository against PostgreSQL using pgx directly — no
// ORM — following the exact pattern established by
// internal/ponport/postgres.PONPortRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// AccessInterfaceRepository implements
// accessinterface.AccessInterfaceRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type AccessInterfaceRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ accessinterface.AccessInterfaceRepository = (*AccessInterfaceRepository)(nil)

// NewAccessInterfaceRepository builds an AccessInterfaceRepository.
func NewAccessInterfaceRepository(db database.Querier, clock clock.Clock, ids id.Generator) *AccessInterfaceRepository {
	return &AccessInterfaceRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves an AccessInterface by ID, or an apperror.KindNotFound
// error if none exists.
func (r *AccessInterfaceRepository) Get(ctx context.Context, interfaceID uuid.UUID) (accessinterface.AccessInterface, error) {
	const query = `
		SELECT id, pon_port_id, technology, name, status, description, created_at, updated_at
		FROM access_interfaces
		WHERE id = $1
	`

	a, err := scanAccessInterface(r.db.QueryRow(ctx, query, interfaceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessinterface.AccessInterface{}, accessInterfaceNotFound(interfaceID)
		}
		return accessinterface.AccessInterface{}, translateError("get access interface", err)
	}
	return a, nil
}

// List returns every AccessInterface, ordered by name for stable,
// human-useful output — the same reasoning
// internal/accessnetwork/postgres.AccessNetworkRepository.List gives for
// its own ordering.
func (r *AccessInterfaceRepository) List(ctx context.Context) ([]accessinterface.AccessInterface, error) {
	const query = `
		SELECT id, pon_port_id, technology, name, status, description, created_at, updated_at
		FROM access_interfaces
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list access interfaces", err)
	}
	defer rows.Close()

	interfaces := []accessinterface.AccessInterface{}
	for rows.Next() {
		a, err := scanAccessInterface(rows)
		if err != nil {
			return nil, translateError("scan access interface row", err)
		}
		interfaces = append(interfaces, a)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list access interfaces", err)
	}

	return interfaces, nil
}

// Create inserts a and returns the persisted record.
//
// As with PONPortRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input
// AccessInterface for those fields are ignored. A PONPortID that does not
// reference an existing PON port fails with an apperror.KindConflict
// error (see translateError).
func (r *AccessInterfaceRepository) Create(ctx context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	const query = `
		INSERT INTO access_interfaces (id, pon_port_id, technology, name, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, pon_port_id, technology, name, status, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanAccessInterface(r.db.QueryRow(ctx, query,
		r.ids.New(), a.PONPortID, string(a.Technology), a.Name, string(a.Status), a.Description, now))
	if err != nil {
		return accessinterface.AccessInterface{}, translateError("create access interface", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the AccessInterface identified
// by a.ID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input AccessInterface
// contained.
func (r *AccessInterfaceRepository) Update(ctx context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	const query = `
		UPDATE access_interfaces
		SET pon_port_id = $1, technology = $2, name = $3, status = $4, description = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, pon_port_id, technology, name, status, description, created_at, updated_at
	`

	updated, err := scanAccessInterface(r.db.QueryRow(ctx, query,
		a.PONPortID, string(a.Technology), a.Name, string(a.Status), a.Description, r.clock.Now(), a.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessinterface.AccessInterface{}, accessInterfaceNotFound(a.ID)
		}
		return accessinterface.AccessInterface{}, translateError("update access interface", err)
	}
	return updated, nil
}

// Delete removes the AccessInterface identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *AccessInterfaceRepository) Delete(ctx context.Context, interfaceID uuid.UUID) error {
	const query = `DELETE FROM access_interfaces WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, interfaceID)
	if err != nil {
		return translateError("delete access interface", err)
	}
	if tag.RowsAffected() == 0 {
		return accessInterfaceNotFound(interfaceID)
	}
	return nil
}

func accessInterfaceNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("access interface %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanAccessInterface backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccessInterface(row rowScanner) (accessinterface.AccessInterface, error) {
	var (
		a          accessinterface.AccessInterface
		technology string
		status     string
	)
	err := row.Scan(&a.ID, &a.PONPortID, &technology, &a.Name, &status, &a.Description, &a.CreatedAt, &a.UpdatedAt)
	a.Technology = accessinterface.Technology(technology)
	a.Status = accessinterface.Status(status)
	return a, err
}
