// Package engine implements the Workflow Engine's orchestration layer:
// driving a single WorkflowInstance through execution against
// internal/plugin's capability-driven registry. It is a direct
// generalization of the former internal/provisioning/engine, with one
// deliberate change in dependency shape: where that engine depended on
// the raw ProvisioningRepository and duplicated start/succeed/fail
// persistence itself, this engine depends on internal/workflow/service's
// transition methods directly. That service layer now also writes an
// event.Event on every transition (docs/02-DESIGN-PRINCIPLES.md,
// principle 10), and duplicating start/succeed/fail here would mean
// Execute silently skips that event trail — going through the same
// transition methods every other caller uses keeps there being exactly
// one place a WorkflowInstance's status changes, and exactly one place
// events get written for it.
package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/plugin"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
)

// Engine drives a WorkflowInstance through execution.
type Engine interface {
	// Execute runs the WorkflowInstance identified by instanceID to
	// completion (Succeeded or Failed) and returns nil on success or the
	// error that caused it to fail.
	Execute(ctx context.Context, instanceID uuid.UUID) error
}

// transitioner is the seam DefaultEngine depends on instead of the
// concrete *workflowservice.Service, so engine tests can exercise
// orchestration against a fake without a real repository, clock, or
// event store. Its shape mirrors the three transition methods Execute
// actually calls.
type transitioner interface {
	Start(ctx context.Context, id uuid.UUID) (workflow.Instance, error)
	Succeed(ctx context.Context, id uuid.UUID) (workflow.Instance, error)
	Fail(ctx context.Context, id uuid.UUID, errorMessage string) (workflow.Instance, error)
}

// DefaultEngine is Engine's one implementation.
type DefaultEngine struct {
	instances transitioner
	services  service.ServiceRepository
	equipment serviceequipment.ServiceEquipmentRepository
	registry  plugin.Registry
	clock     clock.Clock
}

var _ Engine = (*DefaultEngine)(nil)

// NewDefaultEngine builds a DefaultEngine.
func NewDefaultEngine(
	instances transitioner,
	services service.ServiceRepository,
	equipment serviceequipment.ServiceEquipmentRepository,
	registry plugin.Registry,
	clock clock.Clock,
) *DefaultEngine {
	return &DefaultEngine{instances: instances, services: services, equipment: equipment, registry: registry, clock: clock}
}

// serviceStatusAfter maps a plugin.Capability to the service.ServiceStatus
// the target Service should move to once every equipment call for it has
// succeeded. This is a core business rule -- "suspending succeeded means
// the Service is now Suspended" -- not vendor-specific behavior, so it
// lives in the engine rather than in a Plugin: per CLAUDE.md's Plugin
// Philosophy, a plugin reports whether its own action succeeded and
// never reaches back into core domain state itself.
func serviceStatusAfter(capability plugin.Capability) (service.ServiceStatus, bool) {
	switch capability {
	case plugin.ProvisionService, plugin.ReprovisionService, plugin.ResumeService:
		return service.ServiceStatusActive, true
	case plugin.SuspendService:
		return service.ServiceStatusSuspended, true
	case plugin.DisconnectService:
		return service.ServiceStatusDisconnected, true
	case plugin.SynchronizeService:
		// Synchronizing reconciles configuration without changing the
		// Service's own lifecycle state.
		return "", false
	default:
		return "", false
	}
}

// Execute implements Engine. It performs, in order:
//
//  1. Start the instance (Pending -> Running). If the instance is not
//     Pending, Start returns an apperror.KindConflict error and no
//     Service, equipment, or plugin is ever touched.
//  2. Look up the instance's Definition (see workflow.Definitions) to
//     find which plugin.Capability to invoke.
//  3. Load the target Service and every active ServiceEquipment record
//     for it.
//  4. For each active equipment item, resolve the plugin.Plugin
//     registered for the Capability and call its Execute.
//  5. Apply the Capability's effect on the Service's own Status (see
//     serviceStatusAfter) -- e.g. a successful SuspendService moves the
//     Service to Suspended. This is the step that makes execution
//     visible on the Service itself, not just on the WorkflowInstance.
//  6. If every step above succeeds, transition to Succeeded. Any failure
//     from step 3 onward transitions to Failed, recording the failure's
//     message, and returns that error.
//
// Execution stops at the first failure (fail-fast) — there is no queue
// or retry loop here (see docs/05-WORKFLOW-ENGINE.md section 16, an
// explicit future enhancement), so continuing after a failure would not
// change the outcome, only spend more time reaching it.
func (e *DefaultEngine) Execute(ctx context.Context, instanceID uuid.UUID) error {
	instance, err := e.instances.Start(ctx, instanceID)
	if err != nil {
		return err
	}

	definition, ok := workflow.Definitions[instance.DefinitionName]
	if !ok {
		// Unreachable in practice: instance.DefinitionName was already
		// validated (see workflow.Instance.Validate) before this instance
		// was ever persisted. Handled explicitly anyway, the same as
		// every other exhaustive check in this codebase.
		return e.fail(ctx, instanceID, fmt.Errorf("engine: unrecognized workflow definition %q", instance.DefinitionName))
	}

	svc, err := e.services.Get(ctx, instance.ServiceID)
	if err != nil {
		return e.fail(ctx, instanceID, err)
	}

	activeEquipment, err := e.equipment.ListActiveByServiceID(ctx, instance.ServiceID)
	if err != nil {
		return e.fail(ctx, instanceID, err)
	}

	for _, eq := range activeEquipment {
		p, ok := e.registry.Resolve(definition.Capability)
		if !ok {
			return e.fail(ctx, instanceID, fmt.Errorf(
				"no plugin registered for capability %s (equipment %s)", definition.Capability, eq.ID))
		}

		if _, err := p.Execute(ctx, definition.Capability, plugin.Resource{Service: svc, Equipment: eq}); err != nil {
			return e.fail(ctx, instanceID, fmt.Errorf("plugin %q: %w", p.Name(), err))
		}
	}

	if err := e.applyServiceStatus(ctx, svc, definition.Capability); err != nil {
		return e.fail(ctx, instanceID, err)
	}

	_, err = e.instances.Succeed(ctx, instanceID)
	return err
}

// applyServiceStatus updates svc's own Status (and the matching lifecycle
// timestamp) once every equipment call for capability has succeeded, per
// serviceStatusAfter. Capabilities with no defined Service-level effect
// (e.g. SynchronizeService) leave svc untouched.
func (e *DefaultEngine) applyServiceStatus(ctx context.Context, svc service.Service, capability plugin.Capability) error {
	status, ok := serviceStatusAfter(capability)
	if !ok {
		return nil
	}

	now := e.clock.Now()
	svc.Status = status
	switch status {
	case service.ServiceStatusActive:
		svc.ActivatedAt = &now
	case service.ServiceStatusSuspended:
		svc.SuspendedAt = &now
	case service.ServiceStatusDisconnected:
		svc.DisconnectedAt = &now
	}

	_, err := e.services.Update(ctx, svc)
	return err
}

// fail transitions the instance to Failed, recording cause's message,
// and returns cause so the original caller sees why execution failed —
// unless persisting the Failed transition itself fails, in which case
// that persistence error is returned instead, the same reasoning the
// former provisioning engine's own fail method documented: a instance
// stuck unreadably in Running is a more urgent problem than the
// already-known original cause.
func (e *DefaultEngine) fail(ctx context.Context, instanceID uuid.UUID, cause error) error {
	if _, err := e.instances.Fail(ctx, instanceID, cause.Error()); err != nil {
		return err
	}
	return cause
}
