// Package postgres implements the Workflow domain's Repository against
// PostgreSQL using pgx directly — no ORM — a direct port of the former
// internal/provisioning/postgres.
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
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
)

// Repository implements workflow.Repository against PostgreSQL.
type Repository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ workflow.Repository = (*Repository)(nil)

// NewRepository builds a Repository.
func NewRepository(db database.Querier, clock clock.Clock, ids id.Generator) *Repository {
	return &Repository{db: db, clock: clock, ids: ids}
}

// Get retrieves a WorkflowInstance by ID, or an apperror.KindNotFound
// error if none exists.
func (r *Repository) Get(ctx context.Context, instanceID uuid.UUID) (workflow.Instance, error) {
	const query = `
		SELECT id, definition_name, service_id, requested_by_user_id, status, retry_count,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM workflow_instances
		WHERE id = $1
	`

	i, err := scanInstance(r.db.QueryRow(ctx, query, instanceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return workflow.Instance{}, instanceNotFound(instanceID)
		}
		return workflow.Instance{}, translateError("get workflow instance", err)
	}
	return i, nil
}

// List returns every WorkflowInstance, ordered by created_at.
func (r *Repository) List(ctx context.Context) ([]workflow.Instance, error) {
	const query = `
		SELECT id, definition_name, service_id, requested_by_user_id, status, retry_count,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM workflow_instances
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list workflow instances", err)
	}
	defer rows.Close()
	return scanInstances(rows)
}

// ListByServiceID returns every WorkflowInstance for serviceID, ordered
// by created_at.
func (r *Repository) ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]workflow.Instance, error) {
	const query = `
		SELECT id, definition_name, service_id, requested_by_user_id, status, retry_count,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM workflow_instances
		WHERE service_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, translateError("list workflow instances by service", err)
	}
	defer rows.Close()
	return scanInstances(rows)
}

// Create inserts i and returns the persisted record. ID, CreatedAt, and
// UpdatedAt are assigned by the repository.
func (r *Repository) Create(ctx context.Context, i workflow.Instance) (workflow.Instance, error) {
	const query = `
		INSERT INTO workflow_instances (
			id, definition_name, service_id, requested_by_user_id, status, retry_count,
			error_message, started_at, completed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, definition_name, service_id, requested_by_user_id, status, retry_count,
		          error_message, started_at, completed_at, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanInstance(r.db.QueryRow(ctx, query,
		r.ids.New(), i.DefinitionName, i.ServiceID, i.RequestedByUserID, string(i.Status), i.RetryCount,
		i.ErrorMessage, i.StartedAt, i.CompletedAt, now))
	if err != nil {
		return workflow.Instance{}, translateError("create workflow instance", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the WorkflowInstance identified
// by i.ID and returns the persisted record. It performs no
// state-transition enforcement — internal/workflow/service.Service is
// responsible for that before calling Update.
func (r *Repository) Update(ctx context.Context, i workflow.Instance) (workflow.Instance, error) {
	const query = `
		UPDATE workflow_instances
		SET definition_name = $1, service_id = $2, requested_by_user_id = $3, status = $4, retry_count = $5,
		    error_message = $6, started_at = $7, completed_at = $8, updated_at = $9
		WHERE id = $10
		RETURNING id, definition_name, service_id, requested_by_user_id, status, retry_count,
		          error_message, started_at, completed_at, created_at, updated_at
	`

	updated, err := scanInstance(r.db.QueryRow(ctx, query,
		i.DefinitionName, i.ServiceID, i.RequestedByUserID, string(i.Status), i.RetryCount,
		i.ErrorMessage, i.StartedAt, i.CompletedAt, r.clock.Now(), i.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return workflow.Instance{}, instanceNotFound(i.ID)
		}
		return workflow.Instance{}, translateError("update workflow instance", err)
	}
	return updated, nil
}

// Delete removes the WorkflowInstance identified by id.
func (r *Repository) Delete(ctx context.Context, instanceID uuid.UUID) error {
	const query = `DELETE FROM workflow_instances WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, instanceID)
	if err != nil {
		return translateError("delete workflow instance", err)
	}
	if tag.RowsAffected() == 0 {
		return instanceNotFound(instanceID)
	}
	return nil
}

func instanceNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("workflow instance %s not found", id))
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInstance(row rowScanner) (workflow.Instance, error) {
	var (
		i      workflow.Instance
		status string
	)
	err := row.Scan(
		&i.ID, &i.DefinitionName, &i.ServiceID, &i.RequestedByUserID, &status, &i.RetryCount,
		&i.ErrorMessage, &i.StartedAt, &i.CompletedAt, &i.CreatedAt, &i.UpdatedAt,
	)
	i.Status = workflow.Status(status)
	return i, err
}

func scanInstances(rows pgx.Rows) ([]workflow.Instance, error) {
	instances := []workflow.Instance{}
	for rows.Next() {
		i, err := scanInstance(rows)
		if err != nil {
			return nil, translateError("scan workflow instance row", err)
		}
		instances = append(instances, i)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list workflow instances", err)
	}
	return instances, nil
}
