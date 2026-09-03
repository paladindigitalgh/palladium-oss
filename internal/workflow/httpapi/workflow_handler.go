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
// against a fake. Start/Succeed/Fail are deliberately absent: Execute is
// now the one real path through the state machine (see Execute's doc
// comment below), so there is no HTTP route left that needs them
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

// workflowEngine is the seam Execute depends on instead of a concrete
// *engine.DefaultEngine.
type workflowEngine interface {
	Execute(ctx context.Context, instanceID uuid.UUID) error
}

// WorkflowHandler serves the Workflow domain's REST endpoints:
//
//	POST   /api/v1/workflow-instances
//	GET    /api/v1/workflow-instances             (optionally ?service_id=...)
//	GET    /api/v1/workflow-instances/{id}
//	DELETE /api/v1/workflow-instances/{id}
//	POST   /api/v1/workflow-instances/{id}/execute
//	POST   /api/v1/workflow-instances/{id}/cancel
//	POST   /api/v1/workflow-instances/{id}/retry
//
// There is no manual start/succeed/fail route, unlike the former
// provisioning-jobs API: Execute drives the instance through Start,
// every plugin call, and Succeed/Fail itself, so a client only ever
// asks for the outcome it wants ("run this") rather than manually
// puppeteering the state machine one transition at a time.
type WorkflowHandler struct {
	instances workflowService
	engine    workflowEngine
}

// NewWorkflowHandler builds a WorkflowHandler.
func NewWorkflowHandler(instances workflowService, engine workflowEngine) *WorkflowHandler {
	return &WorkflowHandler{instances: instances, engine: engine}
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

// Execute handles POST /api/v1/workflow-instances/{id}/execute: it runs
// the instance to completion synchronously (see engine.Engine.Execute)
// and returns the resulting instance. A failure during execution is
// still reported as an HTTP error (the instance itself is left in its
// Failed state, inspectable via a subsequent GET) rather than a 200 with
// a failure payload — the same "errors are errors" convention every
// other write endpoint in this codebase follows.
func (h *WorkflowHandler) Execute(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.engine.Execute(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.instances.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newInstanceResponse(updated))
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
