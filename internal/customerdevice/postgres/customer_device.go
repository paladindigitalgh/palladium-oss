// Package postgres implements the Customer Device domain's
// CustomerDeviceRepository against PostgreSQL using pgx directly — no
// ORM — mirroring internal/serviceequipment/postgres exactly, one
// domain over.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// CustomerDeviceRepository implements
// customerdevice.CustomerDeviceRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type CustomerDeviceRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ customerdevice.CustomerDeviceRepository = (*CustomerDeviceRepository)(nil)

// NewCustomerDeviceRepository builds a CustomerDeviceRepository.
func NewCustomerDeviceRepository(db database.Querier, clock clock.Clock, ids id.Generator) *CustomerDeviceRepository {
	return &CustomerDeviceRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a CustomerDevice record by ID, or an
// apperror.KindNotFound error if none exists.
func (r *CustomerDeviceRepository) Get(ctx context.Context, recordID uuid.UUID) (customerdevice.CustomerDevice, error) {
	const query = `
		SELECT id, customer_id, device_id, description,
		       attached_at, detached_at, created_at, updated_at
		FROM customer_devices
		WHERE id = $1
	`

	cd, err := scanCustomerDevice(r.db.QueryRow(ctx, query, recordID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return customerdevice.CustomerDevice{}, recordNotFound(recordID)
		}
		return customerdevice.CustomerDevice{}, translateError("get customer device", err)
	}
	return cd, nil
}

// List returns every CustomerDevice record, ordered by created_at for
// stable, human-useful output — the same reasoning
// internal/serviceequipment/postgres.ServiceEquipmentRepository.List
// gives for its own ordering.
func (r *CustomerDeviceRepository) List(ctx context.Context) ([]customerdevice.CustomerDevice, error) {
	const query = `
		SELECT id, customer_id, device_id, description,
		       attached_at, detached_at, created_at, updated_at
		FROM customer_devices
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list customer devices", err)
	}
	defer rows.Close()

	records := []customerdevice.CustomerDevice{}
	for rows.Next() {
		cd, err := scanCustomerDevice(rows)
		if err != nil {
			return nil, translateError("scan customer device row", err)
		}
		records = append(records, cd)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list customer devices", err)
	}

	return records, nil
}

// Create inserts cd and returns the persisted record. As with
// ServiceEquipmentRepository.Create, the repository assigns ID,
// CreatedAt, and UpdatedAt itself — any values already set on the input
// for those fields are ignored. A CustomerID or DeviceID that does not
// reference an existing row fails with an apperror.KindConflict error.
//
// This method does not enforce the active-assignment-uniqueness or
// retired-device business rules — the repository trusts its caller, the
// same as every other repository in this codebase.
// customerdevice/service.CustomerDeviceService checks those before ever
// calling this.
func (r *CustomerDeviceRepository) Create(ctx context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	const query = `
		INSERT INTO customer_devices (
			id, customer_id, device_id, description,
			attached_at, detached_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, customer_id, device_id, description,
		          attached_at, detached_at, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanCustomerDevice(r.db.QueryRow(ctx, query,
		r.ids.New(), cd.CustomerID, cd.DeviceID, cd.Description,
		cd.AttachedAt, cd.DetachedAt, now))
	if err != nil {
		return customerdevice.CustomerDevice{}, translateError("create customer device", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the CustomerDevice identified
// by cd.ID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist. CreatedAt cannot be altered through this
// method. Like Create, this method does not itself enforce any business
// rule — see Create's doc comment.
func (r *CustomerDeviceRepository) Update(ctx context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	const query = `
		UPDATE customer_devices
		SET customer_id = $1, device_id = $2, description = $3,
		    attached_at = $4, detached_at = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, customer_id, device_id, description,
		          attached_at, detached_at, created_at, updated_at
	`

	updated, err := scanCustomerDevice(r.db.QueryRow(ctx, query,
		cd.CustomerID, cd.DeviceID, cd.Description,
		cd.AttachedAt, cd.DetachedAt, r.clock.Now(), cd.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return customerdevice.CustomerDevice{}, recordNotFound(cd.ID)
		}
		return customerdevice.CustomerDevice{}, translateError("update customer device", err)
	}
	return updated, nil
}

// GetActiveByDeviceID returns the active (detached_at IS NULL)
// CustomerDevice attachment for deviceID, or an apperror.KindNotFound
// error if the Device has none. Backs CustomerDeviceService's
// active-assignment-uniqueness and detach-blocking checks.
//
// LIMIT 1 documents the caller's expectation directly in the SQL, the
// same reasoning
// internal/serviceequipment/postgres.ServiceEquipmentRepository.GetActiveByDeviceID's
// own doc comment gives: at most one active attachment per Device is a
// business rule enforced by CustomerDeviceService, not a database
// constraint, so nothing here guarantees only one row could ever match.
func (r *CustomerDeviceRepository) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (customerdevice.CustomerDevice, error) {
	const query = `
		SELECT id, customer_id, device_id, description,
		       attached_at, detached_at, created_at, updated_at
		FROM customer_devices
		WHERE device_id = $1 AND detached_at IS NULL
		LIMIT 1
	`

	cd, err := scanCustomerDevice(r.db.QueryRow(ctx, query, deviceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return customerdevice.CustomerDevice{}, apperror.NotFound(
				fmt.Sprintf("no active customer device attachment for device %s", deviceID))
		}
		return customerdevice.CustomerDevice{}, translateError("get active customer device by device", err)
	}
	return cd, nil
}

func recordNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("customer device %s not found", id))
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows, mirroring
// internal/serviceequipment/postgres.rowScanner.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanCustomerDevice(row rowScanner) (customerdevice.CustomerDevice, error) {
	var cd customerdevice.CustomerDevice
	err := row.Scan(
		&cd.ID, &cd.CustomerID, &cd.DeviceID, &cd.Description,
		&cd.AttachedAt, &cd.DetachedAt, &cd.CreatedAt, &cd.UpdatedAt,
	)
	return cd, err
}
