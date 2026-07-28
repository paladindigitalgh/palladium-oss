// Package engine implements the Provisioning Engine: the orchestration
// layer that will eventually coordinate provisioning across multiple
// external systems (see internal/provisioning/connectors). This
// milestone builds the orchestration flow only — it does not execute
// provisioning. Execute drives a single ProvisioningJob through exactly
// the steps goal 4 describes: load the job, verify it is Pending,
// transition it to Running, load the Service and its active Service
// Equipment, call the connector method matching the job's Operation for
// each active equipment item, and transition the job to Succeeded or
// Failed depending on whether every call succeeded.
//
// There is deliberately no background execution, no retry loop, no
// queue, and no scheduling here (see this milestone's explicit scope):
// Execute is a plain synchronous function call. Whatever calls it — a
// future HTTP handler, a future worker, a test — blocks until it
// returns, the same as any other method in this codebase. Adding any of
// those things is a real future milestone, not implied by anything here.
package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/connectors"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// Engine drives a ProvisioningJob through execution. It is an interface,
// with DefaultEngine as its one implementation here, for the same reason
// every injected dependency in this codebase is (see e.g.
// provisioning.ProvisioningRepository's own doc comment): it lets
// anything that depends on Engine — a future HTTP handler, most likely —
// be tested against a fake instead of a real database and real
// connectors, and it keeps the concrete orchestration logic replaceable
// without changing anything that calls Execute.
type Engine interface {
	// Execute runs the ProvisioningJob identified by jobID to completion
	// (Succeeded or Failed) and returns nil on success or the error that
	// caused it to fail. See DefaultEngine.Execute for the exact steps.
	Execute(ctx context.Context, jobID uuid.UUID) error
}

// DefaultEngine is Engine's one implementation.
//
// It depends on the raw repositories for Provisioning, Service, and
// Service Equipment — not on their business-logic service layers (e.g.
// provisioning/service.ProvisioningService) — because goal 5 names
// exactly these dependencies, and because DefaultEngine's own state
// transitions (Pending -> Running -> Succeeded/Failed) are a small,
// fixed subset of the full state machine
// provisioning/service.ProvisioningService enforces for every
// caller-driven transition. Reusing ProvisioningStatus.CanTransitionTo —
// the domain package's own transition table — keeps DefaultEngine's
// transition logic and ProvisioningService's transition logic answering
// "is this move allowed" from the same single source of truth, even
// though the two do not share a code path for applying it. A future
// milestone could unify them (e.g. by having DefaultEngine call
// ProvisioningService instead of the repository directly); this
// milestone follows goal 5's dependency list as given rather than
// second-guessing it.
type DefaultEngine struct {
	jobs      provisioning.ProvisioningRepository
	services  service.ServiceRepository
	equipment serviceequipment.ServiceEquipmentRepository
	registry  connectors.Registry
	clock     clock.Clock
}

// NewDefaultEngine builds a DefaultEngine.
func NewDefaultEngine(
	jobs provisioning.ProvisioningRepository,
	services service.ServiceRepository,
	equipment serviceequipment.ServiceEquipmentRepository,
	registry connectors.Registry,
	clock clock.Clock,
) *DefaultEngine {
	return &DefaultEngine{
		jobs:      jobs,
		services:  services,
		equipment: equipment,
		registry:  registry,
		clock:     clock,
	}
}

var _ Engine = (*DefaultEngine)(nil)

// Execute implements Engine. It performs, in order:
//
//  1. Load the ProvisioningJob. A NotFound here propagates unchanged —
//     there is nothing to transition if the job does not exist.
//  2. Verify the job can move to Running (in practice: is it Pending —
//     see ProvisioningStatus.CanTransitionTo). If not, Execute returns an
//     apperror.KindConflict error without ever loading the Service,
//     loading equipment, or calling a connector — goal 6's explicit "no
//     connector is called when the job is not Pending" requirement.
//  3. Transition the job to Running and persist that immediately, before
//     doing anything else. This is a real, separate write — not merged
//     into the final Succeeded/Failed write — so a caller inspecting the
//     job mid-execution (e.g. via GET /provisioning-jobs/{id}) sees
//     Running rather than a stale Pending.
//  4. Load the related Service.
//  5. Load every active ServiceEquipment record for that Service (see
//     ServiceEquipmentRepository.ListActiveByServiceID).
//  6. For each active equipment item, look up the Connector registered
//     for its Role and call the one Connector method matching the job's
//     Operation.
//  7. If every call succeeds, transition the job to Succeeded. If any
//     step from 4 onward fails — loading the Service, loading equipment,
//     an unregistered Role, or a Connector method itself — transition
//     the job to Failed, recording the failure's message as
//     ErrorMessage, and return that error.
//
// Every active equipment item participates in every operation; Execute
// does not try to guess which equipment types care about which
// Operation. That is deliberate: deciding "a WiFi access point has
// nothing to do for Suspend" is exactly the kind of vendor- and
// equipment-specific judgment CLAUDE.md's Plugin Philosophy assigns to
// plugins, not core — a Connector that has nothing to do for a given
// operation can simply implement that method as a no-op returning nil.
// A Service with no active equipment at all is not a failure either:
// Execute loads zero items, calls no connector, and succeeds — there is
// nothing in this milestone's scope suggesting "no equipment yet" should
// block a job.
//
// Execution stops at the first Connector failure (fail-fast), not after
// attempting every equipment item and aggregating errors — the simplest
// behavior consistent with "no retries, no queues": there is no
// mechanism here to retry or resume a partially-executed job, so running
// remaining connectors after one has already failed would not change the
// job's outcome, only spend more time reaching it.
func (e *DefaultEngine) Execute(ctx context.Context, jobID uuid.UUID) error {
	job, err := e.jobs.Get(ctx, jobID)
	if err != nil {
		return err
	}

	if !job.Status.CanTransitionTo(provisioning.ProvisioningStatusRunning) {
		return apperror.Conflict(fmt.Sprintf(
			"cannot execute provisioning job %s: status is %s, want %s",
			jobID, job.Status, provisioning.ProvisioningStatusPending))
	}

	job, err = e.start(ctx, job)
	if err != nil {
		return err
	}

	svc, err := e.services.Get(ctx, job.ServiceID)
	if err != nil {
		return e.fail(ctx, job, err)
	}

	activeEquipment, err := e.equipment.ListActiveByServiceID(ctx, job.ServiceID)
	if err != nil {
		return e.fail(ctx, job, err)
	}

	for _, eq := range activeEquipment {
		connector, ok := e.registry.Get(eq.Role)
		if !ok {
			return e.fail(ctx, job, fmt.Errorf(
				"no connector registered for equipment role %s (device %s)", eq.Role, eq.DeviceID))
		}

		req := connectors.Request{Service: svc, Equipment: eq}
		if err := e.call(ctx, connector, job.Operation, req); err != nil {
			return e.fail(ctx, job, fmt.Errorf("connector %q: %w", connector.Name(), err))
		}
	}

	return e.succeed(ctx, job)
}

// call dispatches to the one Connector method matching op. This is the
// one place in this package that maps a ProvisioningOperation to a
// Connector method — everywhere else, Connector is used through its
// interface, never switched on by name.
func (e *DefaultEngine) call(ctx context.Context, c connectors.Connector, op provisioning.ProvisioningOperation, req connectors.Request) error {
	switch op {
	case provisioning.ProvisioningOperationProvision:
		return c.Provision(ctx, req)
	case provisioning.ProvisioningOperationReprovision:
		return c.Reprovision(ctx, req)
	case provisioning.ProvisioningOperationSuspend:
		return c.Suspend(ctx, req)
	case provisioning.ProvisioningOperationResume:
		return c.Resume(ctx, req)
	case provisioning.ProvisioningOperationDisconnect:
		return c.Disconnect(ctx, req)
	case provisioning.ProvisioningOperationSynchronize:
		return c.Synchronize(ctx, req)
	default:
		// Unreachable in practice: job.Operation was already validated
		// (see provisioning.ProvisioningJob.Validate) before this job was
		// ever persisted. Handled explicitly anyway rather than silently
		// doing nothing, on the same principle as every other exhaustive
		// switch in this codebase over a closed enum.
		return fmt.Errorf("engine: unrecognized provisioning operation %q", op)
	}
}

// start transitions job to Running and persists it, stamping StartedAt
// with the current time.
func (e *DefaultEngine) start(ctx context.Context, job provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	job.Status = provisioning.ProvisioningStatusRunning
	now := e.clock.Now()
	job.StartedAt = &now

	if err := job.Validate(); err != nil {
		return provisioning.ProvisioningJob{}, err
	}
	return e.jobs.Update(ctx, job)
}

// succeed transitions job to Succeeded and persists it, stamping
// CompletedAt with the current time.
func (e *DefaultEngine) succeed(ctx context.Context, job provisioning.ProvisioningJob) error {
	job.Status = provisioning.ProvisioningStatusSucceeded
	now := e.clock.Now()
	job.CompletedAt = &now

	if err := job.Validate(); err != nil {
		return err
	}
	if _, err := e.jobs.Update(ctx, job); err != nil {
		return err
	}
	return nil
}

// fail transitions job to Failed, records cause's message as
// ErrorMessage, stamps CompletedAt, and returns cause so the original
// caller sees why execution failed.
//
// If persisting the Failed transition itself fails, that persistence
// error is returned instead of cause. This is a deliberate choice, not
// an oversight: a job stuck unreadably in Running because its Failed
// transition could not be written is a more urgent problem than the
// original connector failure that triggered it, and returning the
// persistence error is what surfaces that to the caller rather than
// masking it behind the (by comparison, already-known) original cause.
func (e *DefaultEngine) fail(ctx context.Context, job provisioning.ProvisioningJob, cause error) error {
	job.Status = provisioning.ProvisioningStatusFailed
	now := e.clock.Now()
	job.CompletedAt = &now
	message := cause.Error()
	job.ErrorMessage = &message

	if err := job.Validate(); err != nil {
		return err
	}
	if _, err := e.jobs.Update(ctx, job); err != nil {
		return err
	}
	return cause
}
