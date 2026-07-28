// Package service is the Provisioning domain's business logic layer. It
// sits between the HTTP layer and the repository layer: HTTP handlers
// never call a repository directly (see internal/provisioning/httpapi),
// and repositories never validate or otherwise reason about business
// rules (see internal/provisioning/postgres, which trusts its caller) —
// this is where those two responsibilities meet. It mirrors
// internal/serviceequipment/service exactly, including that package's
// note on why nesting a package named "service" inside a domain package
// whose name is unrelated (internal/provisioning/service) is the
// expected, consistent result of this codebase's per-domain layering
// convention.
//
// ProvisioningService is the first business logic layer in this codebase
// to depend on clock.Clock. Every other domain's service layer
// deliberately does not (see e.g.
// internal/serviceequipment/service.ServiceEquipmentService's doc
// comment): CreatedAt/UpdatedAt are persistence bookkeeping the
// repository already owns, and no other domain's business rules needed
// to reason about "now". Here, StartedAt and CompletedAt are different in
// kind — they are not "when was this row last written," they are "at
// what moment did this specific state transition happen," which is a
// business fact tied to which of Start/Succeed/Fail/Cancel was actually
// called, not something a generic Update() could infer or a repository
// could stamp on its own. That is what makes this an intentional
// exception rather than an inconsistency.
//
// This package also does not expose a generic Update method, unlike
// every prior domain's service layer (e.g.
// internal/service/service.ServiceService.Update). Goal 7 asks this
// layer to "enforce provisioning state transitions" against the exact
// transition table in provisioning.ProvisioningStatus.CanTransitionTo,
// and a generic Update(ctx, job) taking an arbitrary caller-supplied
// ProvisioningJob would have to reconstruct "which transition is this"
// by diffing against whatever is currently persisted — fragile, and it
// would let a caller change ServiceID, Operation, or RetryCount as a side
// effect of what should be a pure status transition. Start, Succeed,
// Fail, Cancel, and Retry below each correspond to exactly one edge in
// the transition table, take only the parameters that transition
// actually needs, and are the only way a ProvisioningJob's Status changes
// once created. See ProvisioningRepository.Update's own doc comment: the
// repository still has a generic Update, because it is pure persistence
// with no opinion about state — these methods are what calls it.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// ProvisioningService is the Provisioning domain's business logic.
//
// It depends on provisioning.ProvisioningRepository and clock.Clock — see
// this package's doc comment for why the clock dependency is a
// deliberate exception to the pattern every other domain's service layer
// follows.
type ProvisioningService struct {
	jobs  provisioning.ProvisioningRepository
	clock clock.Clock
}

// NewProvisioningService builds a ProvisioningService.
func NewProvisioningService(jobs provisioning.ProvisioningRepository, clock clock.Clock) *ProvisioningService {
	return &ProvisioningService{jobs: jobs, clock: clock}
}

// Get retrieves a ProvisioningJob by ID.
func (s *ProvisioningService) Get(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return s.jobs.Get(ctx, id)
}

// List returns every ProvisioningJob.
func (s *ProvisioningService) List(ctx context.Context) ([]provisioning.ProvisioningJob, error) {
	return s.jobs.List(ctx)
}

// ListByServiceID returns every ProvisioningJob for serviceID.
func (s *ProvisioningService) ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]provisioning.ProvisioningJob, error) {
	return s.jobs.ListByServiceID(ctx, serviceID)
}

// Create validates j and, if valid, persists it as a new job at the
// start of its lifecycle.
//
// Status, RetryCount, ErrorMessage, StartedAt, and CompletedAt are all
// forced here, overwriting whatever the caller supplied: every
// ProvisioningJob begins at ProvisioningStatusPending with a RetryCount
// of zero and no lifecycle timestamps or error recorded yet — that is
// simply what "just created" means for this state machine. Letting a
// caller create a job that claims to already be Succeeded, or that
// starts with a nonzero RetryCount, would let them skip the very
// transitions this service exists to enforce. This mirrors how every
// repository in this codebase already ignores a caller-supplied ID or
// CreatedAt on Create (see e.g. ServiceRepository.Create's doc comment);
// this is that same "identity and lifecycle metadata are not the
// caller's to set" rule, applied one layer up because here it is a
// business rule about the state machine, not raw persistence metadata.
func (s *ProvisioningService) Create(ctx context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	j.Status = provisioning.ProvisioningStatusPending
	j.RetryCount = 0
	j.ErrorMessage = nil
	j.StartedAt = nil
	j.CompletedAt = nil

	if err := j.Validate(); err != nil {
		return provisioning.ProvisioningJob{}, err
	}
	return s.jobs.Create(ctx, j)
}

// Delete removes the ProvisioningJob identified by id.
func (s *ProvisioningService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.jobs.Delete(ctx, id)
}

// Start transitions the ProvisioningJob identified by id from Pending to
// Running, stamping StartedAt with the current time.
func (s *ProvisioningService) Start(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return s.transition(ctx, id, provisioning.ProvisioningStatusRunning, func(j *provisioning.ProvisioningJob) {
		now := s.clock.Now()
		j.StartedAt = &now
	})
}

// Succeed transitions the ProvisioningJob identified by id from Running
// to Succeeded, stamping CompletedAt with the current time.
func (s *ProvisioningService) Succeed(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return s.transition(ctx, id, provisioning.ProvisioningStatusSucceeded, func(j *provisioning.ProvisioningJob) {
		now := s.clock.Now()
		j.CompletedAt = &now
	})
}

// Fail transitions the ProvisioningJob identified by id from Running to
// Failed, stamping CompletedAt with the current time and recording
// errorMessage.
func (s *ProvisioningService) Fail(ctx context.Context, id uuid.UUID, errorMessage string) (provisioning.ProvisioningJob, error) {
	return s.transition(ctx, id, provisioning.ProvisioningStatusFailed, func(j *provisioning.ProvisioningJob) {
		now := s.clock.Now()
		j.CompletedAt = &now
		j.ErrorMessage = &errorMessage
	})
}

// Cancel transitions the ProvisioningJob identified by id from Pending or
// Running to Cancelled, stamping CompletedAt with the current time.
func (s *ProvisioningService) Cancel(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return s.transition(ctx, id, provisioning.ProvisioningStatusCancelled, func(j *provisioning.ProvisioningJob) {
		now := s.clock.Now()
		j.CompletedAt = &now
	})
}

// Retry transitions the ProvisioningJob identified by id from Failed back
// to Pending, incrementing RetryCount. This is the one place RetryCount
// changes: goal 7 explicitly asks this package not to implement retry
// logic ("RetryCount is simply stored"), so there is no backoff, no
// automatic re-scheduling, and no cap on how many times Retry may be
// called — it only ever records that a retry was requested, by
// incrementing a counter, and moves the job back to the front of its
// lifecycle so a future Start can run it again.
//
// ErrorMessage, StartedAt, and CompletedAt from the failed attempt are
// deliberately left untouched here rather than cleared: nothing in this
// milestone's scope asks for that history to be erased, and the next
// Start/Succeed/Fail call will naturally overwrite StartedAt/CompletedAt
// (and ErrorMessage, if it fails again) with the new attempt's own
// values. Clearing them preemptively would destroy information a caller
// might still want to see (e.g. "why did the previous attempt fail")
// between the Retry and the next Start.
func (s *ProvisioningService) Retry(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return s.transition(ctx, id, provisioning.ProvisioningStatusPending, func(j *provisioning.ProvisioningJob) {
		j.RetryCount++
	})
}

// transition is the shared implementation behind Start, Succeed, Fail,
// Cancel, and Retry: fetch the current job, confirm the move to target is
// allowed by provisioning.ProvisioningStatus.CanTransitionTo, apply the
// transition-specific side effect via mutate, validate the result, and
// persist it. Centralizing this here — rather than repeating
// fetch/check/validate/persist five times — is what keeps each of those
// five public methods reading as "one line naming its transition."
//
// A NotFound from jobs.Get propagates unchanged. A transition that
// CanTransitionTo disallows becomes an apperror.KindConflict, the same
// Kind this codebase uses for every other "the request conflicts with
// the current state of the data" situation (see e.g.
// internal/serviceequipment/service's active-assignment-uniqueness
// check) — attempting to Start an already-Succeeded job is exactly that
// kind of conflict, not a malformed request (KindInvalid) or a missing
// resource (KindNotFound).
func (s *ProvisioningService) transition(
	ctx context.Context,
	id uuid.UUID,
	target provisioning.ProvisioningStatus,
	mutate func(*provisioning.ProvisioningJob),
) (provisioning.ProvisioningJob, error) {
	job, err := s.jobs.Get(ctx, id)
	if err != nil {
		return provisioning.ProvisioningJob{}, err
	}

	if !job.Status.CanTransitionTo(target) {
		return provisioning.ProvisioningJob{}, apperror.Conflict(
			fmt.Sprintf("cannot transition provisioning job from %s to %s", job.Status, target))
	}

	job.Status = target
	mutate(&job)

	if err := job.Validate(); err != nil {
		return provisioning.ProvisioningJob{}, err
	}
	return s.jobs.Update(ctx, job)
}
