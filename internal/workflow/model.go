// Package workflow implements Palladium's Workflow Engine (v1): the
// orchestration layer described in docs/05-WORKFLOW-ENGINE.md. A
// WorkflowInstance is a single execution of a named Definition (see
// definition.go) against a Service — the direct generalization of the
// former internal/provisioning package, replaying the same lifecycle
// against internal/plugin's capability-driven Plugin interface instead
// of a fixed six-method Connector.
//
// This package holds only the domain model, field validation, and the
// repository interface — no SQL, no migrations, no HTTP CRUD.
package workflow

import (
	"time"

	"github.com/google/uuid"
)

// Instance is a single execution of a named Definition against a
// Service.
type Instance struct {
	ID                uuid.UUID
	DefinitionName    string
	ServiceID         uuid.UUID
	RequestedByUserID *uuid.UUID
	Status            Status
	RetryCount        int
	ErrorMessage      *string
	StartedAt         *time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
