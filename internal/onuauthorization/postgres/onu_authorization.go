// Package postgres implements the ONU Authorization domain's
// onuauthorization.Repository against PostgreSQL using pgx directly — no
// ORM — following the exact pattern
// internal/accessattachment/postgres.AccessAttachmentRepository
// establishes, narrowed to the three methods onuauthorization.Repository
// actually declares.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// OnuAuthorizationRepository implements onuauthorization.Repository
// against PostgreSQL. See internal/inventory/postgres/site.go for the
// reasoning behind depending on database.Querier and injecting
// clock/ids, which is not repeated here.
type OnuAuthorizationRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ onuauthorization.Repository = (*OnuAuthorizationRepository)(nil)

// NewOnuAuthorizationRepository builds an OnuAuthorizationRepository.
func NewOnuAuthorizationRepository(db database.Querier, clock clock.Clock, ids id.Generator) *OnuAuthorizationRepository {
	return &OnuAuthorizationRepository{db: db, clock: clock, ids: ids}
}

// Create inserts authorization and returns the persisted record. As with
// every other repository in this codebase, ID, CreatedAt, and UpdatedAt
// are assigned here — any values already set on the input are ignored.
// A DeviceID or OLTID that does not reference an existing row fails with
// an apperror.KindConflict error (see translateError).
func (r *OnuAuthorizationRepository) Create(ctx context.Context, authorization onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error) {
	const query = `
		INSERT INTO onu_authorizations (
			id, device_id, olt_id, interface, authorized_at, deauthorized_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, device_id, olt_id, interface, authorized_at,
		          deauthorized_at, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanOnuAuthorization(r.db.QueryRow(ctx, query,
		r.ids.New(), authorization.DeviceID, authorization.OLTID, authorization.Interface,
		authorization.AuthorizedAt, authorization.DeauthorizedAt, now))
	if err != nil {
		return onuauthorization.OnuAuthorization{}, translateError("create onu authorization", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the OnuAuthorization identified
// by authorization.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist. DeviceID and OLTID
// are included even though nothing in this codebase changes them today,
// mirroring accessattachment.postgres.AccessAttachmentRepository.Update's
// own reasoning: this method overwrites every mutable column, not a
// hand-picked subset, so a future caller cannot silently forget one.
func (r *OnuAuthorizationRepository) Update(ctx context.Context, authorization onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error) {
	const query = `
		UPDATE onu_authorizations
		SET device_id = $1, olt_id = $2, interface = $3, authorized_at = $4,
		    deauthorized_at = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, device_id, olt_id, interface, authorized_at,
		          deauthorized_at, created_at, updated_at
	`

	updated, err := scanOnuAuthorization(r.db.QueryRow(ctx, query,
		authorization.DeviceID, authorization.OLTID, authorization.Interface,
		authorization.AuthorizedAt, authorization.DeauthorizedAt, r.clock.Now(), authorization.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return onuauthorization.OnuAuthorization{}, authorizationNotFound(authorization.ID)
		}
		return onuauthorization.OnuAuthorization{}, translateError("update onu authorization", err)
	}
	return updated, nil
}

// GetActiveByDeviceID returns the active (deauthorized_at IS NULL)
// OnuAuthorization for deviceID, or an apperror.KindNotFound error if it
// has none. LIMIT 1 mirrors
// accessattachment.postgres.AccessAttachmentRepository.GetActiveByServiceEquipmentID's
// own reasoning: at most one active authorization per Device is this
// domain's expectation, not a database constraint, so this returns a
// real (if arbitrary) active row rather than erroring if that were ever
// violated.
func (r *OnuAuthorizationRepository) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (onuauthorization.OnuAuthorization, error) {
	const query = `
		SELECT id, device_id, olt_id, interface, authorized_at,
		       deauthorized_at, created_at, updated_at
		FROM onu_authorizations
		WHERE device_id = $1 AND deauthorized_at IS NULL
		LIMIT 1
	`

	a, err := scanOnuAuthorization(r.db.QueryRow(ctx, query, deviceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return onuauthorization.OnuAuthorization{}, apperror.NotFound(
				fmt.Sprintf("no active onu authorization for device %s", deviceID))
		}
		return onuauthorization.OnuAuthorization{}, translateError("get active onu authorization by device", err)
	}
	return a, nil
}

func authorizationNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("onu authorization %s not found", id))
}

// rowScanner is satisfied by pgx.Row (QueryRow), the only shape this
// repository's three methods ever scan from.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanOnuAuthorization(row rowScanner) (onuauthorization.OnuAuthorization, error) {
	var a onuauthorization.OnuAuthorization
	err := row.Scan(
		&a.ID, &a.DeviceID, &a.OLTID, &a.Interface, &a.AuthorizedAt,
		&a.DeauthorizedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}
