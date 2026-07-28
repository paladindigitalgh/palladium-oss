// Package postgres implements the PON Port domain's PONPortRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/olt/postgres.OLTRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// PONPortRepository implements ponport.PONPortRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type PONPortRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ ponport.PONPortRepository = (*PONPortRepository)(nil)

// NewPONPortRepository builds a PONPortRepository.
func NewPONPortRepository(db database.Querier, clock clock.Clock, ids id.Generator) *PONPortRepository {
	return &PONPortRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a PONPort by ID, or an apperror.KindNotFound error if
// none exists.
func (r *PONPortRepository) Get(ctx context.Context, portID uuid.UUID) (ponport.PONPort, error) {
	const query = `
		SELECT id, olt_id, port_number, description, created_at, updated_at
		FROM pon_ports
		WHERE id = $1
	`

	p, err := scanPONPort(r.db.QueryRow(ctx, query, portID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ponport.PONPort{}, ponPortNotFound(portID)
		}
		return ponport.PONPort{}, translateError("get pon port", err)
	}
	return p, nil
}

// List returns every PONPort, ordered by port_number for stable,
// human-useful output — unlike Access Network or OLT, a PON port has no
// name column to order by, and port_number is the natural, already-real
// ordering an operator would expect (lowest port first).
func (r *PONPortRepository) List(ctx context.Context) ([]ponport.PONPort, error) {
	const query = `
		SELECT id, olt_id, port_number, description, created_at, updated_at
		FROM pon_ports
		ORDER BY port_number
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list pon ports", err)
	}
	defer rows.Close()

	ports := []ponport.PONPort{}
	for rows.Next() {
		p, err := scanPONPort(rows)
		if err != nil {
			return nil, translateError("scan pon port row", err)
		}
		ports = append(ports, p)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list pon ports", err)
	}

	return ports, nil
}

// Create inserts p and returns the persisted record.
//
// As with OLTRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input PONPort for
// those fields are ignored. An OLTID that does not reference an existing
// OLT fails with an apperror.KindConflict error (see translateError).
func (r *PONPortRepository) Create(ctx context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	const query = `
		INSERT INTO pon_ports (id, olt_id, port_number, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, olt_id, port_number, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanPONPort(r.db.QueryRow(ctx, query,
		r.ids.New(), p.OLTID, p.PortNumber, p.Description, now))
	if err != nil {
		return ponport.PONPort{}, translateError("create pon port", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (OLTID, PortNumber, Description)
// of the PONPort identified by p.ID and returns the persisted record, or
// an apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input PONPort contained.
func (r *PONPortRepository) Update(ctx context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	const query = `
		UPDATE pon_ports
		SET olt_id = $1, port_number = $2, description = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, olt_id, port_number, description, created_at, updated_at
	`

	updated, err := scanPONPort(r.db.QueryRow(ctx, query,
		p.OLTID, p.PortNumber, p.Description, r.clock.Now(), p.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ponport.PONPort{}, ponPortNotFound(p.ID)
		}
		return ponport.PONPort{}, translateError("update pon port", err)
	}
	return updated, nil
}

// Delete removes the PONPort identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *PONPortRepository) Delete(ctx context.Context, portID uuid.UUID) error {
	const query = `DELETE FROM pon_ports WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, portID)
	if err != nil {
		return translateError("delete pon port", err)
	}
	if tag.RowsAffected() == 0 {
		return ponPortNotFound(portID)
	}
	return nil
}

func ponPortNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("pon port %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanPONPort backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanPONPort(row rowScanner) (ponport.PONPort, error) {
	var p ponport.PONPort
	err := row.Scan(&p.ID, &p.OLTID, &p.PortNumber, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
