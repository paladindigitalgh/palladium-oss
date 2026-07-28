package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// provisioningService is the seam ProvisioningHandler depends on instead
// of a concrete *service.ProvisioningService — the same reasoning
// internal/serviceequipment/httpapi's serviceEquipmentService interface
// documents: it lets handler tests exercise HTTP behavior (status codes,
// JSON shapes, routing, error mapping) against a fake, with no real
// service, repository, or database involved. Unexported for the same
// reason serviceEquipmentService is: Go interfaces are satisfied
// structurally, so nothing outside this package needs to name it.
//
// Its shape mirrors ProvisioningService's own public methods exactly,
// including Start/Succeed/Fail/Cancel/Retry in place of a generic
// Update — see internal/provisioning/service's package doc comment for
// why that service has no generic Update to depend on here.
type provisioningService interface {
	Get(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error)
	List(ctx context.Context) ([]provisioning.ProvisioningJob, error)
	ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]provisioning.ProvisioningJob, error)
	Create(ctx context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Start(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error)
	Succeed(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error)
	Fail(ctx context.Context, id uuid.UUID, errorMessage string) (provisioning.ProvisioningJob, error)
	Cancel(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error)
	Retry(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error)
}

// ProvisioningHandler serves the Provisioning REST endpoints:
//
//	POST   /api/v1/provisioning-jobs
//	GET    /api/v1/provisioning-jobs             (optionally ?service_id=...)
//	GET    /api/v1/provisioning-jobs/{id}
//	DELETE /api/v1/provisioning-jobs/{id}
//	POST   /api/v1/provisioning-jobs/{id}/start
//	POST   /api/v1/provisioning-jobs/{id}/succeed
//	POST   /api/v1/provisioning-jobs/{id}/fail
//	POST   /api/v1/provisioning-jobs/{id}/cancel
//	POST   /api/v1/provisioning-jobs/{id}/retry
//
// There is deliberately no PUT /provisioning-jobs/{id}: a generic
// "replace this job with whatever JSON you sent" endpoint has no honest
// meaning for a state machine, since it would have to guess which of
// Start/Succeed/Fail/Cancel/Retry a caller intended from a diff (see
// internal/provisioning/service's package doc comment for the full
// reasoning). Action sub-routes name the intent directly instead — the
// same pattern REST APIs commonly use for state transitions on a
// resource (e.g. cancelling or rerunning a CI run) — and each maps to
// exactly one ProvisioningService method, which is what keeps every
// handler below a two-or-three-line decode/delegate/translate.
//
// It depends only on provisioningService — never a repository or the
// concrete service type directly — so it has no knowledge of PostgreSQL,
// SQL, or any storage technology, and no awareness that a state machine
// even exists beyond "call this method, translate whatever error comes
// back." That is ProvisioningService's job.
type ProvisioningHandler struct {
	jobs provisioningService
}

// NewProvisioningHandler builds a ProvisioningHandler.
func NewProvisioningHandler(jobs provisioningService) *ProvisioningHandler {
	return &ProvisioningHandler{jobs: jobs}
}

// Create handles POST /api/v1/provisioning-jobs.
//
// RequestedByUserID is set from the authenticated caller's own identity
// (auth.ClaimsFromContext), never from the request body — see
// provisioningJobCreateRequest's doc comment in dto.go for why. This
// route sits behind auth.Middleware in production (see
// internal/server/router.go), so Claims are always present here; if they
// were ever absent, RequestedByUserID is simply left nil, which is a
// valid state for it (see provisioning.ProvisioningJob's doc comment),
// not an error condition this handler needs to guard against.
func (h *ProvisioningHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req provisioningJobCreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	job := provisioning.ProvisioningJob{
		ServiceID: req.ServiceID,
		Operation: provisioning.ProvisioningOperation(req.Operation),
	}
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		job.RequestedByUserID = &claims.UserID
	}

	created, err := h.jobs.Create(r.Context(), job)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newProvisioningJobResponse(created))
}

// List handles GET /api/v1/provisioning-jobs. An optional ?service_id=
// query parameter narrows the result to ProvisioningService.ListByServiceID
// instead of List — goal 6's ListByServiceID exposed as a query filter on
// the same collection endpoint, rather than a separate route, since it is
// the same resource collection either way, just optionally scoped.
func (h *ProvisioningHandler) List(w http.ResponseWriter, r *http.Request) {
	if raw := r.URL.Query().Get("service_id"); raw != "" {
		serviceID, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, apperror.Invalid("service_id must be a valid UUID"))
			return
		}

		jobs, err := h.jobs.ListByServiceID(r.Context(), serviceID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, newProvisioningJobListResponse(jobs))
		return
	}

	jobs, err := h.jobs.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningJobListResponse(jobs))
}

// Get handles GET /api/v1/provisioning-jobs/{id}.
func (h *ProvisioningHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	j, err := h.jobs.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningJobResponse(j))
}

// Delete handles DELETE /api/v1/provisioning-jobs/{id}.
func (h *ProvisioningHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.jobs.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Start handles POST /api/v1/provisioning-jobs/{id}/start.
func (h *ProvisioningHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.jobs.Start(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningJobResponse(updated))
}

// Succeed handles POST /api/v1/provisioning-jobs/{id}/succeed.
func (h *ProvisioningHandler) Succeed(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.jobs.Succeed(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningJobResponse(updated))
}

// Fail handles POST /api/v1/provisioning-jobs/{id}/fail.
func (h *ProvisioningHandler) Fail(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req provisioningJobFailRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.jobs.Fail(r.Context(), id, req.ErrorMessage)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningJobResponse(updated))
}

// Cancel handles POST /api/v1/provisioning-jobs/{id}/cancel.
func (h *ProvisioningHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.jobs.Cancel(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningJobResponse(updated))
}

// Retry handles POST /api/v1/provisioning-jobs/{id}/retry.
func (h *ProvisioningHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.jobs.Retry(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningJobResponse(updated))
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
