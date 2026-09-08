// Package worker implements the Workflow Engine's job queue: a single
// background goroutine that polls for Pending WorkflowInstances and
// drives each one through internal/workflow/engine.Engine.Execute.
//
// This is what makes workflow execution asynchronous
// (docs/05-WORKFLOW-ENGINE.md's "Job queue" future enhancement,
// TASKS.md Phase 7): a client that creates a WorkflowInstance no longer
// also has to call an execute endpoint inline in the same request — see
// internal/workflow/httpapi's package doc comment for why that endpoint
// was removed. Instead, this Worker notices the new Pending row on its
// next poll and runs it.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
)

// pendingSource is the seam Worker depends on instead of the full
// workflow.Repository, so worker tests can exercise the poll loop
// against a fake without a real repository — the same narrow-interface
// pattern internal/workflow/engine.transitioner already establishes.
type pendingSource interface {
	NextPending(ctx context.Context) (workflow.Instance, bool, error)
}

// engine is the seam Worker depends on instead of a concrete
// *engine.DefaultEngine, mirroring internal/workflow/httpapi's own
// workflowEngine seam.
type engine interface {
	Execute(ctx context.Context, instanceID uuid.UUID) error
}

// Worker polls for Pending WorkflowInstances and executes them one at a
// time. There is no concurrency here — a second Pending instance found
// on one poll simply gets picked up on the very next poll, once the
// current one finishes — which keeps this package's only real job
// (notice new work, run it) a single small loop, matching this
// codebase's "avoid unnecessary abstractions" bias. If plugin calls ever
// become slow enough that this serialization is a real throughput
// problem, that is the point to introduce a bounded worker pool, not
// before.
type Worker struct {
	pending      pendingSource
	engine       engine
	pollInterval time.Duration
	logger       *slog.Logger
}

// New builds a Worker.
func New(pending pendingSource, engine engine, pollInterval time.Duration, logger *slog.Logger) *Worker {
	return &Worker{pending: pending, engine: engine, pollInterval: pollInterval, logger: logger}
}

// Run polls until ctx is cancelled, at which point it returns promptly —
// mirroring internal/httpserver.Server.Run's own ctx-driven lifecycle, so
// cmd/server/main.go can start and stop both the same way.
func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("workflow worker started", "poll_interval", w.pollInterval)
	defer w.logger.Info("workflow worker stopped")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		instance, ok, err := w.pending.NextPending(ctx)
		if err != nil {
			w.logger.Error("workflow worker: poll for pending instance failed", "error", err)
			if !w.sleep(ctx) {
				return
			}
			continue
		}

		if !ok {
			if !w.sleep(ctx) {
				return
			}
			continue
		}

		// Execute already persists the instance's own Succeeded/Failed
		// transition (and the Event that records it — see
		// internal/workflow/engine's package doc comment) before
		// returning, so a failure here needs nothing more from the
		// worker than a log line: there is no result to report back to,
		// and no caller blocked waiting on this call the way the old
		// synchronous HTTP handler was.
		if err := w.engine.Execute(ctx, instance.ID); err != nil {
			w.logger.Error("workflow worker: execute failed", "workflow_instance_id", instance.ID, "error", err)
		}
	}
}

// sleep waits for w.pollInterval, or returns false immediately if ctx is
// cancelled first — the only way Run's loop exits.
func (w *Worker) sleep(ctx context.Context) bool {
	timer := time.NewTimer(w.pollInterval)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
