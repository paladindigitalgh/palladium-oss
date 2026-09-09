package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// DeviceRepository implements inventory.DeviceRepository against
// PostgreSQL. It follows SiteRepository (site.go) exactly; see that file
// for the reasoning behind depending on database.Querier and injecting
// clock/ids, which is not repeated here. The only structural difference
// from the other three repositories in this package is the extra plain
// columns (manufacturer, model, serial_number, asset_tag, status); the
// pattern — assign identity/timestamps in Create, protect CreatedAt in
// Update, translate errors — is identical.
type DeviceRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ inventory.DeviceRepository = (*DeviceRepository)(nil)

// NewDeviceRepository builds a DeviceRepository.
func NewDeviceRepository(db database.Querier, clock clock.Clock, ids id.Generator) *DeviceRepository {
	return &DeviceRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Device by ID, or an apperror.KindNotFound error if none
// exists.
func (r *DeviceRepository) Get(ctx context.Context, deviceID uuid.UUID) (inventory.Device, error) {
	const query = `
		SELECT id, rack_id, name, description, device_model_id, serial_number,
		       asset_tag, status, created_at, updated_at
		FROM devices
		WHERE id = $1
	`

	device, err := scanDevice(r.db.QueryRow(ctx, query, deviceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Device{}, deviceNotFound(deviceID)
		}
		return inventory.Device{}, translateError("get device", err)
	}
	return device, nil
}

// GetBySerialNumber retrieves a Device by its exact SerialNumber, or an
// apperror.KindNotFound error if none exists. See
// inventory.DeviceRepository.GetBySerialNumber's own doc comment on why
// no ORDER BY/LIMIT tie-breaking is needed here despite the column
// having no unique constraint.
func (r *DeviceRepository) GetBySerialNumber(ctx context.Context, serialNumber string) (inventory.Device, error) {
	const query = `
		SELECT id, rack_id, name, description, device_model_id, serial_number,
		       asset_tag, status, created_at, updated_at
		FROM devices
		WHERE serial_number = $1
		LIMIT 1
	`

	device, err := scanDevice(r.db.QueryRow(ctx, query, serialNumber))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Device{}, apperror.NotFound(fmt.Sprintf("device with serial number %q not found", serialNumber))
		}
		return inventory.Device{}, translateError("get device by serial number", err)
	}
	return device, nil
}

// List returns every Device, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *DeviceRepository) List(ctx context.Context) ([]inventory.Device, error) {
	const query = `
		SELECT id, rack_id, name, description, device_model_id, serial_number,
		       asset_tag, status, created_at, updated_at
		FROM devices
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list devices", err)
	}
	defer rows.Close()

	devices := []inventory.Device{}
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, translateError("scan device row", err)
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list devices", err)
	}

	return devices, nil
}

// Create inserts device and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt, and
// UpdatedAt itself; any values already set on the input Device for those
// fields are ignored. RackID may be nil (see inventory.Device); a non-nil
// RackID that does not reference an existing Rack fails with an
// apperror.KindConflict error (see translateError), and likewise a
// DeviceModelID that does not reference an existing DeviceModel.
func (r *DeviceRepository) Create(ctx context.Context, device inventory.Device) (inventory.Device, error) {
	const query = `
		INSERT INTO devices (id, rack_id, name, description, device_model_id,
		                      serial_number, asset_tag, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id, rack_id, name, description, device_model_id, serial_number,
		          asset_tag, status, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanDevice(r.db.QueryRow(ctx, query,
		r.ids.New(), device.RackID, device.Name, device.Description,
		device.DeviceModelID, device.SerialNumber, device.AssetTag,
		string(device.Status), now))
	if err != nil {
		return inventory.Device{}, translateError("create device", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the Device identified by
// device.ID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist.
//
// CreatedAt cannot be altered through this method, for the same reason as
// SiteRepository.Update. RackID is treated as mutable, for the same reason
// BuildingRepository.Update treats SiteID as mutable — this is also how a
// Device transitions from unracked (nil RackID) to installed.
func (r *DeviceRepository) Update(ctx context.Context, device inventory.Device) (inventory.Device, error) {
	const query = `
		UPDATE devices
		SET rack_id = $1, name = $2, description = $3, device_model_id = $4,
		    serial_number = $5, asset_tag = $6, status = $7, updated_at = $8
		WHERE id = $9
		RETURNING id, rack_id, name, description, device_model_id, serial_number,
		          asset_tag, status, created_at, updated_at
	`

	updated, err := scanDevice(r.db.QueryRow(ctx, query,
		device.RackID, device.Name, device.Description,
		device.DeviceModelID, device.SerialNumber, device.AssetTag,
		string(device.Status), r.clock.Now(), device.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inventory.Device{}, deviceNotFound(device.ID)
		}
		return inventory.Device{}, translateError("update device", err)
	}
	return updated, nil
}

// Delete removes the Device identified by id, or returns an
// apperror.KindNotFound error if it does not exist. Device is a leaf in
// the inventory hierarchy itself, but it is still referenced from
// outside it: service_equipment.device_id ON DELETE RESTRICT rejects
// this delete with an apperror.KindConflict error (see translateError)
// whenever this Device has ever been part of a Service, active or not
// (see internal/serviceequipment).
func (r *DeviceRepository) Delete(ctx context.Context, deviceID uuid.UUID) error {
	const query = `DELETE FROM devices WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, deviceID)
	if err != nil {
		return translateError("delete device", err)
	}
	if tag.RowsAffected() == 0 {
		return deviceNotFound(deviceID)
	}
	return nil
}

func deviceNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("device %s not found", id))
}

func scanDevice(row rowScanner) (inventory.Device, error) {
	var (
		device inventory.Device
		status string
	)
	err := row.Scan(&device.ID, &device.RackID, &device.Name, &device.Description,
		&device.DeviceModelID, &device.SerialNumber, &device.AssetTag,
		&status, &device.CreatedAt, &device.UpdatedAt)
	device.Status = inventory.DeviceStatus(status)
	return device, err
}
