package workflow

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists WorkflowInstances. Create and Update return the
// persisted entity, matching every other repository in this codebase.
// The repository enforces no business rules, including no state-
// transition checks — internal/workflow/service.Service is where those
// live.
type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (Instance, error)
	List(ctx context.Context) ([]Instance, error)
	ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]Instance, error)
	Create(ctx context.Context, i Instance) (Instance, error)
	Update(ctx context.Context, i Instance) (Instance, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
