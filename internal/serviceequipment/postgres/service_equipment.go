// Package postgres implements the Service Equipment domain's
// ServiceEquipmentRepository against PostgreSQL using pgx directly — no
// ORM — following the exact pattern established by
// internal/service/postgres.ServiceRepository (the closest precedent:
// two required foreign keys to sibling domains).
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
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// ServiceEquipmentRepository implements
// serviceequipment.ServiceEquipmentRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type ServiceEquipmentRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ serviceequipment.ServiceEquipmentRepository = (*ServiceEquipmentRepository)(nil)

// NewServiceEquipmentRepository builds a ServiceEquipmentRepository.
func NewServiceEquipmentRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ServiceEquipmentRepository {
	return &ServiceEquipmentRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a ServiceEquipment record by ID, or an
// apperror.KindNotFound error if none exists.
func (r *ServiceEquipmentRepository) Get(ctx context.Context, equipmentID uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	const query = `
		SELECT id, service_id, device_id, role, description,
		       installed_at, removed_at, created_at, updated_at
		FROM service_equipment
		WHERE id = $1
	`

	e, err := scanServiceEquipment(r.db.QueryRow(ctx, query, equipmentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return serviceequipment.ServiceEquipment{}, equipmentNotFound(equipmentID)
		}
		return serviceequipment.ServiceEquipment{}, translateError("get service equipment", err)
	}
	return e, nil
}

// List returns every ServiceEquipment record, ordered by created_at for
// stable, human-useful output — the same reasoning
// internal/service/postgres.ServiceRepository.List gives for its own
// ordering: an equipment assignment has no name column to order by.
func (r *ServiceEquipmentRepository) List(ctx context.Context) ([]serviceequipment.ServiceEquipment, error) {
	const query = `
		SELECT id, service_id, device_id, role, description,
		       installed_at, removed_at, created_at, updated_at
		FROM service_equipment
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list service equipment", err)
	}
	defer rows.Close()

	equipment := []serviceequipment.ServiceEquipment{}
	for rows.Next() {
		e, err := scanServiceEquipment(rows)
		if err != nil {
			return nil, translateError("scan service equipment row", err)
		}
		equipment = append(equipment, e)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list service equipment", err)
	}

	return equipment, nil
}

// Create inserts e and returns the persisted record.
//
// As with ServiceRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input
// ServiceEquipment for those fields are ignored. A ServiceID or DeviceID
// that does not reference an existing row fails with an
// apperror.KindConflict error (see translateError).
//
// This method does not enforce the active-assignment-uniqueness business
// rule (goal 2) — the repository trusts its caller, the same as every
// other repository in this codebase. ServiceEquipmentService (see
// internal/serviceequipment/service) checks GetActiveByDeviceID before
// ever calling this.
func (r *ServiceEquipmentRepository) Create(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	const query = `
		INSERT INTO service_equipment (
			id, service_id, device_id, role, description,
			installed_at, removed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING id, service_id, device_id, role, description,
		          installed_at, removed_at, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanServiceEquipment(r.db.QueryRow(ctx, query,
		r.ids.New(), e.ServiceID, e.DeviceID, string(e.Role), e.Description,
		e.InstalledAt, e.RemovedAt, now))
	if err != nil {
		return serviceequipment.ServiceEquipment{}, translateError("create service equipment", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the ServiceEquipment identified
// by e.ID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input ServiceEquipment
// contained. Like Create, this method does not itself enforce the
// active-assignment-uniqueness rule — see Create's doc comment.
func (r *ServiceEquipmentRepository) Update(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	const query = `
		UPDATE service_equipment
		SET service_id = $1, device_id = $2, role = $3, description = $4,
		    installed_at = $5, removed_at = $6, updated_at = $7
		WHERE id = $8
		RETURNING id, service_id, device_id, role, description,
		          installed_at, removed_at, created_at, updated_at
	`

	updated, err := scanServiceEquipment(r.db.QueryRow(ctx, query,
		e.ServiceID, e.DeviceID, string(e.Role), e.Description,
		e.InstalledAt, e.RemovedAt, r.clock.Now(), e.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return serviceequipment.ServiceEquipment{}, equipmentNotFound(e.ID)
		}
		return serviceequipment.ServiceEquipment{}, translateError("update service equipment", err)
	}
	return updated, nil
}

// Delete removes the ServiceEquipment identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *ServiceEquipmentRepository) Delete(ctx context.Context, equipmentID uuid.UUID) error {
	const query = `DELETE FROM service_equipment WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, equipmentID)
	if err != nil {
		return translateError("delete service equipment", err)
	}
	if tag.RowsAffected() == 0 {
		return equipmentNotFound(equipmentID)
	}
	return nil
}

// GetActiveByDeviceID returns the active (removed_at IS NULL)
// ServiceEquipment assignment for deviceID, or an apperror.KindNotFound
// error if the Device has none. This is the query goal 4 asks for by
// name, backing ServiceEquipmentService's active-assignment-uniqueness
// check (see internal/serviceequipment/service).
//
// LIMIT 1 documents the caller's expectation directly in the SQL: at most
// one active assignment per Device is a business rule enforced by
// ServiceEquipmentService, not a database constraint (see this
// migration's own comment on that choice in
// database/migrations/00014_serviceequipment_service_equipment.sql), so
// nothing here guarantees only one row could ever match. LIMIT 1 makes
// this query behave predictably regardless — it still returns a real, if
// arbitrary, existing active row rather than erroring — without requiring
// the schema to enforce what this milestone deliberately keeps as
// application logic.
func (r *ServiceEquipmentRepository) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	const query = `
		SELECT id, service_id, device_id, role, description,
		       installed_at, removed_at, created_at, updated_at
		FROM service_equipment
		WHERE device_id = $1 AND removed_at IS NULL
		LIMIT 1
	`

	e, err := scanServiceEquipment(r.db.QueryRow(ctx, query, deviceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return serviceequipment.ServiceEquipment{}, apperror.NotFound(
				fmt.Sprintf("no active service equipment assignment for device %s", deviceID))
		}
		return serviceequipment.ServiceEquipment{}, translateError("get active service equipment by device", err)
	}
	return e, nil
}

func equipmentNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("service equipment %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanServiceEquipment backs Get/Create/Update/GetActiveByDeviceID and
// List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanServiceEquipment(row rowScanner) (serviceequipment.ServiceEquipment, error) {
	var (
		e    serviceequipment.ServiceEquipment
		role string
	)
	err := row.Scan(
		&e.ID, &e.ServiceID, &e.DeviceID, &role, &e.Description,
		&e.InstalledAt, &e.RemovedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	e.Role = serviceequipment.EquipmentRole(role)
	return e, err
}
