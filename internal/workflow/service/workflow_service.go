// Package service is the Workflow domain's business logic layer,
// mirroring internal/provisioning/service's former state machine
// exactly, with one addition: every transition also writes an
// event.Event (docs/02-DESIGN-PRINCIPLES.md, principle 10 — "Workflows
// create Events"), which is what populates a Service or Customer
// Workspace's Timeline section for real.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
)

// entityTypeService is the event.Event.EntityType every WorkflowInstance
// transition is recorded against — a WorkflowInstance always acts on
// exactly one Service (see workflow.Instance.ServiceID).
const entityTypeService = "service"

// Service is the Workflow domain's business logic. Like the former
// ProvisioningService, it depends on clock.Clock: StartedAt/CompletedAt
// are business facts tied to which transition was actually called, not
// persistence bookkeeping a repository can stamp on its own.
type Service struct {
	instances workflow.Repository
	events    event.EventRepository
	clock     clock.Clock
}

// New builds a Service.
func New(instances workflow.Repository, events event.EventRepository, clock clock.Clock) *Service {
	return &Service{instances: instances, events: events, clock: clock}
}

// Get retrieves a WorkflowInstance by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (workflow.Instance, error) {
	return s.instances.Get(ctx, id)
}

// List returns every WorkflowInstance.
func (s *Service) List(ctx context.Context) ([]workflow.Instance, error) {
	return s.instances.List(ctx)
}

// ListByServiceID returns every WorkflowInstance for serviceID.
func (s *Service) ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]workflow.Instance, error) {
	return s.instances.ListByServiceID(ctx, serviceID)
}

// Create validates i and persists it as a new instance at the start of
// its lifecycle. Status, RetryCount, ErrorMessage, and the lifecycle
// timestamps are all forced here, overwriting whatever the caller
// supplied — a caller cannot create an instance that claims to already
// be Succeeded, or skip the transitions this service exists to enforce.
func (s *Service) Create(ctx context.Context, i workflow.Instance) (workflow.Instance, error) {
	i.Status = workflow.StatusPending
	i.RetryCount = 0
	i.ErrorMessage = nil
	i.StartedAt = nil
	i.CompletedAt = nil

	if err := i.Validate(); err != nil {
		return workflow.Instance{}, err
	}
	return s.instances.Create(ctx, i)
}

// Delete removes the WorkflowInstance identified by id.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.instances.Delete(ctx, id)
}

// Start transitions the WorkflowInstance identified by id from Pending
// to Running, stamping StartedAt.
func (s *Service) Start(ctx context.Context, id uuid.UUID) (workflow.Instance, error) {
	return s.transition(ctx, id, workflow.StatusRunning, func(i *workflow.Instance) {
		now := s.clock.Now()
		i.StartedAt = &now
	}, "workflow.started", "Workflow started")
}

// Succeed transitions the WorkflowInstance identified by id from Running
// to Succeeded, stamping CompletedAt.
func (s *Service) Succeed(ctx context.Context, id uuid.UUID) (workflow.Instance, error) {
	return s.transition(ctx, id, workflow.StatusSucceeded, func(i *workflow.Instance) {
		now := s.clock.Now()
		i.CompletedAt = &now
	}, "workflow.succeeded", "Workflow succeeded")
}

// Fail transitions the WorkflowInstance identified by id from Running to
// Failed, stamping CompletedAt and recording errorMessage.
func (s *Service) Fail(ctx context.Context, id uuid.UUID, errorMessage string) (workflow.Instance, error) {
	return s.transition(ctx, id, workflow.StatusFailed, func(i *workflow.Instance) {
		now := s.clock.Now()
		i.CompletedAt = &now
		i.ErrorMessage = &errorMessage
	}, "workflow.failed", fmt.Sprintf("Workflow failed: %s", errorMessage))
}

// Cancel transitions the WorkflowInstance identified by id from Pending
// or Running to Cancelled, stamping CompletedAt.
func (s *Service) Cancel(ctx context.Context, id uuid.UUID) (workflow.Instance, error) {
	return s.transition(ctx, id, workflow.StatusCancelled, func(i *workflow.Instance) {
		now := s.clock.Now()
		i.CompletedAt = &now
	}, "workflow.cancelled", "Workflow cancelled")
}

// Retry transitions the WorkflowInstance identified by id from Failed
// back to Pending, incrementing RetryCount. ErrorMessage, StartedAt, and
// CompletedAt from the failed attempt are left untouched — the next
// Start/Succeed/Fail call naturally overwrites them.
func (s *Service) Retry(ctx context.Context, id uuid.UUID) (workflow.Instance, error) {
	return s.transition(ctx, id, workflow.StatusPending, func(i *workflow.Instance) {
		i.RetryCount++
	}, "workflow.retried", "Workflow queued for retry")
}

// transition is the shared implementation behind every public transition
// method: fetch, confirm the move is allowed, apply the transition's side
// effect, validate, persist, and record an Event. A NotFound from
// instances.Get propagates unchanged. A disallowed transition becomes an
// apperror.KindConflict.
func (s *Service) transition(
	ctx context.Context,
	id uuid.UUID,
	target workflow.Status,
	mutate func(*workflow.Instance),
	eventType, eventMessage string,
) (workflow.Instance, error) {
	instance, err := s.instances.Get(ctx, id)
	if err != nil {
		return workflow.Instance{}, err
	}

	if !instance.Status.CanTransitionTo(target) {
		return workflow.Instance{}, apperror.Conflict(
			fmt.Sprintf("cannot transition workflow instance from %s to %s", instance.Status, target))
	}

	instance.Status = target
	mutate(&instance)

	if err := instance.Validate(); err != nil {
		return workflow.Instance{}, err
	}
	updated, err := s.instances.Update(ctx, instance)
	if err != nil {
		return workflow.Instance{}, err
	}

	if _, err := s.events.Create(ctx, event.Event{
		EntityType:  entityTypeService,
		EntityID:    updated.ServiceID,
		Type:        eventType,
		Message:     eventMessage,
		ActorUserID: updated.RequestedByUserID,
		Metadata: map[string]any{
			"workflow_instance_id": updated.ID.String(),
			"definition_name":      updated.DefinitionName,
		},
	}); err != nil {
		return workflow.Instance{}, apperror.Internal("record workflow event", err)
	}

	return updated, nil
}
