package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
)

// workflowService is the seam WorkflowHandler depends on instead of a
// concrete *service.Service, so handler tests can exercise HTTP behavior
// against a fake. Start/Succeed/Fail are deliberately absent: those are
// the engine's job (see internal/workflow/engine and
// internal/workflow/worker), not something an HTTP client drives
// directly.
type workflowService interface {
	Get(ctx context.Context, id uuid.UUID) (workflow.Instance, error)
	List(ctx context.Context) ([]workflow.Instance, error)
	ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]workflow.Instance, error)
	Create(ctx context.Context, i workflow.Instance) (workflow.Instance, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Cancel(ctx context.Context, id uuid.UUID) (workflow.Instance, error)
	Retry(ctx context.Context, id uuid.UUID) (workflow.Instance, error)
}

// WorkflowHandler serves the Workflow domain's REST endpoints:
//
//	POST   /api/v1/workflow-instances
//	GET    /api/v1/workflow-instances             (optionally ?service_id=...)
//	GET    /api/v1/workflow-instances/{id}
//	DELETE /api/v1/workflow-instances/{id}
//	POST   /api/v1/workflow-instances/{id}/cancel
//	POST   /api/v1/workflow-instances/{id}/retry
//
// There used to also be a POST .../{id}/execute route that ran a
// WorkflowInstance to completion synchronously, inline in the HTTP
// request. It is gone: execution is now asynchronous
// (docs/05-WORKFLOW-ENGINE.md's job queue, TASKS.md Phase 7) — creating
// an Instance is enough, since internal/workflow/worker.Worker polls for
// Pending instances and calls engine.Engine.Execute on them itself.
// Keeping the manual route alongside the worker would have reintroduced
// exactly the race internal/workflow/postgres.Repository.NextPending's
// doc comment already flags for a second worker replica, just one layer
// up: a client calling /execute at the same moment the worker polls the
// same Pending instance could both call Start on it, since
// Repository.Update performs no compare-and-swap on status. Removing the
// manual path makes the worker the one and only caller of Execute, which
// avoids that race by construction instead of adding locking to prevent
// it. A client that wants to know when a WorkflowInstance finishes now
// polls GET .../{id} for a terminal status, the same way
// frontend/src/services/workflow/workflowRepository.ts's runWorkflow
// does.
type WorkflowHandler struct {
	instances workflowService
}

// NewWorkflowHandler builds a WorkflowHandler.
func NewWorkflowHandler(instances workflowService) *WorkflowHandler {
	return &WorkflowHandler{instances: instances}
}

// Create handles POST /api/v1/workflow-instances.
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req instanceCreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	instance := workflow.Instance{
		ServiceID:      req.ServiceID,
		DefinitionName: req.DefinitionName,
	}
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		instance.RequestedByUserID = &claims.UserID
	}

	created, err := h.instances.Create(r.Context(), instance)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newInstanceResponse(created))
}

// List handles GET /api/v1/workflow-instances.
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	if raw := r.URL.Query().Get("service_id"); raw != "" {
		serviceID, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, apperror.Invalid("service_id must be a valid UUID"))
			return
		}

		instances, err := h.instances.ListByServiceID(r.Context(), serviceID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, newInstanceListResponse(instances))
		return
	}

	instances, err := h.instances.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newInstanceListResponse(instances))
}

// Get handles GET /api/v1/workflow-instances/{id}.
func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	i, err := h.instances.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newInstanceResponse(i))
}

// Delete handles DELETE /api/v1/workflow-instances/{id}.
func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.instances.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Cancel handles POST /api/v1/workflow-instances/{id}/cancel.
func (h *WorkflowHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.instances.Cancel(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newInstanceResponse(updated))
}

// Retry handles POST /api/v1/workflow-instances/{id}/retry.
func (h *WorkflowHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.instances.Retry(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newInstanceResponse(updated))
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
