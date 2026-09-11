// Package postgres implements the Report domain's Repository against
// PostgreSQL using pgx directly — no ORM — following the pattern
// established by internal/event/postgres. Every method here is a real,
// hand-written SQL join across other domains' tables (see
// internal/report's own package doc comment on why that is this
// package's whole reason to exist); nothing here validates or otherwise
// reasons about business rules, and nothing here ever writes.
package postgres

import (
	"context"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/report"
)

// Repository implements report.Repository against PostgreSQL.
//
// Unlike most repositories in this codebase, this one needs neither
// clock.Clock nor platform/id.Generator: every method here only ever
// reads, so there is no CreatedAt to stamp and no ID to generate.
type Repository struct {
	db database.Querier
}

var _ report.Repository = (*Repository)(nil)

// NewRepository builds a Repository.
func NewRepository(db database.Querier) *Repository {
	return &Repository{db: db}
}

// CustomersWithContacts backs the "Customers & Contacts" report: every
// Customer, LEFT JOINed to Contacts so a Customer with none on file still
// gets exactly one row, every Contact column coalesced to its type's zero
// value (see report.CustomerContactRow's own doc comment) rather than
// left NULL — a LEFT JOIN's unmatched side is NULL column-by-column
// regardless of that column's own NOT NULL constraint on contacts
// itself, so this has to be handled here, not assumed away by the
// schema.
func (r *Repository) CustomersWithContacts(ctx context.Context) ([]report.CustomerContactRow, error) {
	const query = `
		SELECT
			c.id, c.name, c.customer_type, c.status,
			COALESCE(ct.id, '00000000-0000-0000-0000-000000000000'), COALESCE(ct.name, ''),
			COALESCE(ct.role, ''), COALESCE(ct.email, ''), COALESCE(ct.phone, ''), COALESCE(ct.status, '')
		FROM customers c
		LEFT JOIN contacts ct ON ct.customer_id = c.id
		ORDER BY c.name, ct.name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list customers with contacts", err)
	}
	defer rows.Close()

	result := []report.CustomerContactRow{}
	for rows.Next() {
		var row report.CustomerContactRow
		if err := rows.Scan(
			&row.CustomerID, &row.CustomerName, &row.CustomerType, &row.CustomerStatus,
			&row.ContactID, &row.ContactName, &row.ContactRole, &row.ContactEmail, &row.ContactPhone, &row.ContactStatus,
		); err != nil {
			return nil, translateError("scan customer contact row", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list customers with contacts", err)
	}

	return result, nil
}

// Devices backs the "Devices" report: every Device, its resolved
// Manufacturer/Model, its physical location path (LEFT JOINed all the
// way from Rack to Site, since a Device may be unracked — see
// inventory.Device's own RackID doc comment), and whichever Customer
// currently has it.
//
// The LATERAL subquery picks a Customer through either of the two paths
// docs/03-DOMAIN-MODEL.md §9 describes: an active CustomerDevice
// placement (detached_at IS NULL), or an active ServiceEquipment
// assignment (removed_at IS NULL) reached via its Service's Location. It
// is a UNION ALL capped at one row, not two separate LEFT JOINs, so a
// Device that somehow has both active at once still contributes exactly
// one row to this report rather than being duplicated — the placement
// path is listed first only because it is the more direct of the two,
// not because of any ordering guarantee this report promises elsewhere.
func (r *Repository) Devices(ctx context.Context) ([]report.DeviceRow, error) {
	const query = `
		SELECT
			d.id, d.name, d.serial_number, d.asset_tag, d.status,
			dman.name, dm.name,
			COALESCE(s.name, ''), COALESCE(b.name, ''), COALESCE(rm.name, ''), COALESCE(rk.name, ''),
			COALESCE(cust.customer_id, '00000000-0000-0000-0000-000000000000'), COALESCE(cust.customer_name, '')
		FROM devices d
		JOIN device_models dm ON dm.id = d.device_model_id
		JOIN device_manufacturers dman ON dman.id = dm.manufacturer_id
		LEFT JOIN racks rk ON rk.id = d.rack_id
		LEFT JOIN rooms rm ON rm.id = rk.room_id
		LEFT JOIN buildings b ON b.id = rm.building_id
		LEFT JOIN sites s ON s.id = b.site_id
		LEFT JOIN LATERAL (
			SELECT c.id AS customer_id, c.name AS customer_name
			FROM customer_devices cd
			JOIN customers c ON c.id = cd.customer_id
			WHERE cd.device_id = d.id AND cd.detached_at IS NULL
			UNION ALL
			SELECT c.id, c.name
			FROM service_equipment se
			JOIN services sv ON sv.id = se.service_id
			JOIN locations loc ON loc.id = sv.location_id
			JOIN customers c ON c.id = loc.customer_id
			WHERE se.device_id = d.id AND se.removed_at IS NULL
			LIMIT 1
		) cust ON true
		ORDER BY d.name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list devices report", err)
	}
	defer rows.Close()

	result := []report.DeviceRow{}
	for rows.Next() {
		var row report.DeviceRow
		if err := rows.Scan(
			&row.DeviceID, &row.DeviceName, &row.SerialNumber, &row.AssetTag, &row.DeviceStatus,
			&row.Manufacturer, &row.Model,
			&row.SiteName, &row.BuildingName, &row.RoomName, &row.RackName,
			&row.AssignedCustomerID, &row.AssignedCustomerName,
		); err != nil {
			return nil, translateError("scan device report row", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list devices report", err)
	}

	return result, nil
}

// CustomersWithDevices backs the "Customers & Devices" report: every
// currently-active Customer-Device relationship, one row per
// relationship rather than per Device (see report.CustomerDeviceRow's
// own doc comment on how this differs from Devices' single
// "whichever-wins" column above) — a UNION ALL of the same two paths
// Devices resolves through its LATERAL subquery, this time each
// contributing its own row instead of being collapsed into one.
func (r *Repository) CustomersWithDevices(ctx context.Context) ([]report.CustomerDeviceRow, error) {
	const query = `
		SELECT customer_id, customer_name, customer_type, customer_status,
			device_id, device_name, serial_number, manufacturer, model, device_status,
			relationship, service_status, location_name
		FROM (
			SELECT
				c.id AS customer_id, c.name AS customer_name, c.customer_type, c.status AS customer_status,
				d.id AS device_id, d.name AS device_name, d.serial_number, dman.name AS manufacturer, dm.name AS model, d.status AS device_status,
				'Placement' AS relationship, '' AS service_status, '' AS location_name
			FROM customer_devices cd
			JOIN customers c ON c.id = cd.customer_id
			JOIN devices d ON d.id = cd.device_id
			JOIN device_models dm ON dm.id = d.device_model_id
			JOIN device_manufacturers dman ON dman.id = dm.manufacturer_id
			WHERE cd.detached_at IS NULL

			UNION ALL

			SELECT
				c.id, c.name, c.customer_type, c.status,
				d.id, d.name, d.serial_number, dman.name, dm.name, d.status,
				'Service Equipment', sv.status, loc.name
			FROM service_equipment se
			JOIN services sv ON sv.id = se.service_id
			JOIN locations loc ON loc.id = sv.location_id
			JOIN customers c ON c.id = loc.customer_id
			JOIN devices d ON d.id = se.device_id
			JOIN device_models dm ON dm.id = d.device_model_id
			JOIN device_manufacturers dman ON dman.id = dm.manufacturer_id
			WHERE se.removed_at IS NULL
		) relationships
		ORDER BY customer_name, device_name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list customers with devices", err)
	}
	defer rows.Close()

	result := []report.CustomerDeviceRow{}
	for rows.Next() {
		var row report.CustomerDeviceRow
		if err := rows.Scan(
			&row.CustomerID, &row.CustomerName, &row.CustomerType, &row.CustomerStatus,
			&row.DeviceID, &row.DeviceName, &row.SerialNumber, &row.Manufacturer, &row.Model, &row.DeviceStatus,
			&row.Relationship, &row.ServiceStatus, &row.LocationName,
		); err != nil {
			return nil, translateError("scan customer device row", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list customers with devices", err)
	}

	return result, nil
}
