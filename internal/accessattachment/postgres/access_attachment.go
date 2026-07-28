// Package postgres implements the Access Attachment domain's
// AccessAttachmentRepository against PostgreSQL using pgx directly — no
// ORM — following the exact pattern established by
// internal/serviceequipment/postgres.ServiceEquipmentRepository (the
// closest precedent: two required foreign keys to sibling domains, plus
// a GetActiveBy... query backing a service-layer uniqueness rule).
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// AccessAttachmentRepository implements
// accessattachment.AccessAttachmentRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type AccessAttachmentRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ accessattachment.AccessAttachmentRepository = (*AccessAttachmentRepository)(nil)

// NewAccessAttachmentRepository builds an AccessAttachmentRepository.
func NewAccessAttachmentRepository(db database.Querier, clock clock.Clock, ids id.Generator) *AccessAttachmentRepository {
	return &AccessAttachmentRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves an AccessAttachment record by ID, or an
// apperror.KindNotFound error if none exists.
func (r *AccessAttachmentRepository) Get(ctx context.Context, attachmentID uuid.UUID) (accessattachment.AccessAttachment, error) {
	const query = `
		SELECT id, access_interface_id, service_equipment_id, installed_at,
		       removed_at, removal_reason, created_at, updated_at
		FROM access_attachments
		WHERE id = $1
	`

	a, err := scanAccessAttachment(r.db.QueryRow(ctx, query, attachmentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessattachment.AccessAttachment{}, attachmentNotFound(attachmentID)
		}
		return accessattachment.AccessAttachment{}, translateError("get access attachment", err)
	}
	return a, nil
}

// List returns every AccessAttachment record, ordered by created_at for
// stable, human-useful output — the same reasoning
// internal/serviceequipment/postgres.ServiceEquipmentRepository.List
// gives for its own ordering: an attachment has no name column to order
// by.
func (r *AccessAttachmentRepository) List(ctx context.Context) ([]accessattachment.AccessAttachment, error) {
	const query = `
		SELECT id, access_interface_id, service_equipment_id, installed_at,
		       removed_at, removal_reason, created_at, updated_at
		FROM access_attachments
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list access attachments", err)
	}
	defer rows.Close()

	attachments := []accessattachment.AccessAttachment{}
	for rows.Next() {
		a, err := scanAccessAttachment(rows)
		if err != nil {
			return nil, translateError("scan access attachment row", err)
		}
		attachments = append(attachments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list access attachments", err)
	}

	return attachments, nil
}

// Create inserts a and returns the persisted record.
//
// As with ServiceEquipmentRepository.Create, the repository assigns ID,
// CreatedAt, and UpdatedAt itself — any values already set on the input
// AccessAttachment for those fields are ignored. An AccessInterfaceID or
// ServiceEquipmentID that does not reference an existing row fails with
// an apperror.KindConflict error (see translateError).
//
// This method does not enforce the active-attachment-uniqueness business
// rule (goal 2) — the repository trusts its caller, the same as every
// other repository in this codebase. AccessAttachmentService (see
// internal/accessattachment/service) checks
// GetActiveByServiceEquipmentID before ever calling this.
func (r *AccessAttachmentRepository) Create(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	const query = `
		INSERT INTO access_attachments (
			id, access_interface_id, service_equipment_id, installed_at,
			removed_at, removal_reason, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, access_interface_id, service_equipment_id, installed_at,
		          removed_at, removal_reason, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanAccessAttachment(r.db.QueryRow(ctx, query,
		r.ids.New(), a.AccessInterfaceID, a.ServiceEquipmentID,
		a.InstalledAt, a.RemovedAt, a.RemovalReason, now))
	if err != nil {
		return accessattachment.AccessAttachment{}, translateError("create access attachment", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the AccessAttachment
// identified by a.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input AccessAttachment
// contained. Like Create, this method does not itself enforce the
// active-attachment-uniqueness rule — see Create's doc comment.
func (r *AccessAttachmentRepository) Update(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	const query = `
		UPDATE access_attachments
		SET access_interface_id = $1, service_equipment_id = $2, installed_at = $3,
		    removed_at = $4, removal_reason = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, access_interface_id, service_equipment_id, installed_at,
		          removed_at, removal_reason, created_at, updated_at
	`

	updated, err := scanAccessAttachment(r.db.QueryRow(ctx, query,
		a.AccessInterfaceID, a.ServiceEquipmentID, a.InstalledAt,
		a.RemovedAt, a.RemovalReason, r.clock.Now(), a.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessattachment.AccessAttachment{}, attachmentNotFound(a.ID)
		}
		return accessattachment.AccessAttachment{}, translateError("update access attachment", err)
	}
	return updated, nil
}

// Delete removes the AccessAttachment identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *AccessAttachmentRepository) Delete(ctx context.Context, attachmentID uuid.UUID) error {
	const query = `DELETE FROM access_attachments WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, attachmentID)
	if err != nil {
		return translateError("delete access attachment", err)
	}
	if tag.RowsAffected() == 0 {
		return attachmentNotFound(attachmentID)
	}
	return nil
}

// GetActiveByServiceEquipmentID returns the active (removed_at IS NULL)
// AccessAttachment for serviceEquipmentID, or an apperror.KindNotFound
// error if that ServiceEquipment has none. This is the query goal 4
// asks for by name, backing AccessAttachmentService's
// active-attachment-uniqueness check (see
// internal/accessattachment/service).
//
// LIMIT 1 documents the caller's expectation directly in the SQL: at
// most one active attachment per ServiceEquipment is a business rule
// enforced by AccessAttachmentService, not a database constraint (see
// this migration's own comment on that choice in
// database/migrations/00020_accessattachment_access_attachments.sql), so
// nothing here guarantees only one row could ever match. LIMIT 1 makes
// this query behave predictably regardless — it still returns a real,
// if arbitrary, existing active row rather than erroring — without
// requiring the schema to enforce what this milestone deliberately keeps
// as application logic. Mirrors
// serviceequipment.postgres.ServiceEquipmentRepository.GetActiveByDeviceID
// exactly.
func (r *AccessAttachmentRepository) GetActiveByServiceEquipmentID(ctx context.Context, serviceEquipmentID uuid.UUID) (accessattachment.AccessAttachment, error) {
	const query = `
		SELECT id, access_interface_id, service_equipment_id, installed_at,
		       removed_at, removal_reason, created_at, updated_at
		FROM access_attachments
		WHERE service_equipment_id = $1 AND removed_at IS NULL
		LIMIT 1
	`

	a, err := scanAccessAttachment(r.db.QueryRow(ctx, query, serviceEquipmentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accessattachment.AccessAttachment{}, apperror.NotFound(
				fmt.Sprintf("no active access attachment for service equipment %s", serviceEquipmentID))
		}
		return accessattachment.AccessAttachment{}, translateError("get active access attachment by service equipment", err)
	}
	return a, nil
}

func attachmentNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("access attachment %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanAccessAttachment backs Get/Create/Update/GetActiveByServiceEquipmentID
// and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccessAttachment(row rowScanner) (accessattachment.AccessAttachment, error) {
	var a accessattachment.AccessAttachment
	err := row.Scan(
		&a.ID, &a.AccessInterfaceID, &a.ServiceEquipmentID, &a.InstalledAt,
		&a.RemovedAt, &a.RemovalReason, &a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}
