// Package postgres implements the Provisioning domain's
// ProvisioningRepository against PostgreSQL using pgx directly — no ORM —
// following the exact pattern established by
// internal/serviceequipment/postgres.ServiceEquipmentRepository (the
// closest precedent: two foreign keys, one of them nullable).
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
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// ProvisioningRepository implements provisioning.ProvisioningRepository
// against PostgreSQL. See internal/inventory/postgres/site.go for the
// reasoning behind depending on database.Querier and injecting
// clock/ids, which is not repeated here.
type ProvisioningRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ provisioning.ProvisioningRepository = (*ProvisioningRepository)(nil)

// NewProvisioningRepository builds a ProvisioningRepository.
func NewProvisioningRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ProvisioningRepository {
	return &ProvisioningRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a ProvisioningJob by ID, or an apperror.KindNotFound
// error if none exists.
func (r *ProvisioningRepository) Get(ctx context.Context, jobID uuid.UUID) (provisioning.ProvisioningJob, error) {
	const query = `
		SELECT id, service_id, requested_by_user_id, operation, status, retry_count,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM provisioning_jobs
		WHERE id = $1
	`

	j, err := scanProvisioningJob(r.db.QueryRow(ctx, query, jobID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provisioning.ProvisioningJob{}, jobNotFound(jobID)
		}
		return provisioning.ProvisioningJob{}, translateError("get provisioning job", err)
	}
	return j, nil
}

// List returns every ProvisioningJob, ordered by created_at for stable,
// human-useful output — the same reasoning
// internal/service/postgres.ServiceRepository.List gives for its own
// ordering: a provisioning job has no name column to order by, and
// creation order matches how such history is typically reviewed.
func (r *ProvisioningRepository) List(ctx context.Context) ([]provisioning.ProvisioningJob, error) {
	const query = `
		SELECT id, service_id, requested_by_user_id, operation, status, retry_count,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM provisioning_jobs
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list provisioning jobs", err)
	}
	defer rows.Close()

	jobs := []provisioning.ProvisioningJob{}
	for rows.Next() {
		j, err := scanProvisioningJob(rows)
		if err != nil {
			return nil, translateError("scan provisioning job row", err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list provisioning jobs", err)
	}

	return jobs, nil
}

// ListByServiceID returns every ProvisioningJob for serviceID, ordered by
// created_at (oldest first, the same ordering as List) — the natural
// order for reviewing one Service's provisioning history from start to
// most recent.
func (r *ProvisioningRepository) ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]provisioning.ProvisioningJob, error) {
	const query = `
		SELECT id, service_id, requested_by_user_id, operation, status, retry_count,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM provisioning_jobs
		WHERE service_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, translateError("list provisioning jobs by service", err)
	}
	defer rows.Close()

	jobs := []provisioning.ProvisioningJob{}
	for rows.Next() {
		j, err := scanProvisioningJob(rows)
		if err != nil {
			return nil, translateError("scan provisioning job row", err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list provisioning jobs by service", err)
	}

	return jobs, nil
}

// Create inserts j and returns the persisted record.
//
// As with ServiceEquipmentRepository.Create, the repository assigns ID,
// CreatedAt, and UpdatedAt itself — any values already set on the input
// ProvisioningJob for those fields are ignored. A ServiceID or
// RequestedByUserID that does not reference an existing row fails with an
// apperror.KindConflict error (see translateError). The repository does
// not decide what Status, RetryCount, or any other field should be —
// ProvisioningService (see internal/provisioning/service) owns those
// decisions; this method persists exactly the ProvisioningJob it is
// given.
func (r *ProvisioningRepository) Create(ctx context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	const query = `
		INSERT INTO provisioning_jobs (
			id, service_id, requested_by_user_id, operation, status, retry_count,
			error_message, started_at, completed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, service_id, requested_by_user_id, operation, status, retry_count,
		          error_message, started_at, completed_at, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanProvisioningJob(r.db.QueryRow(ctx, query,
		r.ids.New(), j.ServiceID, j.RequestedByUserID, string(j.Operation), string(j.Status), j.RetryCount,
		j.ErrorMessage, j.StartedAt, j.CompletedAt, now))
	if err != nil {
		return provisioning.ProvisioningJob{}, translateError("create provisioning job", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the ProvisioningJob identified
// by j.ID and returns the persisted record, or an apperror.KindNotFound
// error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input ProvisioningJob
// contained. This method performs no state-transition enforcement — it
// persists whatever Status (and RetryCount, ErrorMessage, StartedAt,
// CompletedAt) it is given, trusting its caller completely.
// ProvisioningService's Start/Succeed/Fail/Cancel/Retry methods are the
// only production callers, and each has already checked
// ProvisioningStatus.CanTransitionTo before calling this.
func (r *ProvisioningRepository) Update(ctx context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	const query = `
		UPDATE provisioning_jobs
		SET service_id = $1, requested_by_user_id = $2, operation = $3, status = $4, retry_count = $5,
		    error_message = $6, started_at = $7, completed_at = $8, updated_at = $9
		WHERE id = $10
		RETURNING id, service_id, requested_by_user_id, operation, status, retry_count,
		          error_message, started_at, completed_at, created_at, updated_at
	`

	updated, err := scanProvisioningJob(r.db.QueryRow(ctx, query,
		j.ServiceID, j.RequestedByUserID, string(j.Operation), string(j.Status), j.RetryCount,
		j.ErrorMessage, j.StartedAt, j.CompletedAt, r.clock.Now(), j.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provisioning.ProvisioningJob{}, jobNotFound(j.ID)
		}
		return provisioning.ProvisioningJob{}, translateError("update provisioning job", err)
	}
	return updated, nil
}

// Delete removes the ProvisioningJob identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *ProvisioningRepository) Delete(ctx context.Context, jobID uuid.UUID) error {
	const query = `DELETE FROM provisioning_jobs WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, jobID)
	if err != nil {
		return translateError("delete provisioning job", err)
	}
	if tag.RowsAffected() == 0 {
		return jobNotFound(jobID)
	}
	return nil
}

func jobNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("provisioning job %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanProvisioningJob backs Get/Create/Update and List/ListByServiceID
// alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanProvisioningJob(row rowScanner) (provisioning.ProvisioningJob, error) {
	var (
		j         provisioning.ProvisioningJob
		operation string
		status    string
	)
	err := row.Scan(
		&j.ID, &j.ServiceID, &j.RequestedByUserID, &operation, &status, &j.RetryCount,
		&j.ErrorMessage, &j.StartedAt, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt,
	)
	j.Operation = provisioning.ProvisioningOperation(operation)
	j.Status = provisioning.ProvisioningStatus(status)
	return j, err
}
