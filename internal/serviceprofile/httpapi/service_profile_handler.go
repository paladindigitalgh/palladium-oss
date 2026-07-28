package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
)

// serviceProfileService is the seam ServiceProfileHandler depends on
// instead of a concrete *service.ServiceProfileService — the same
// reasoning internal/catalog/httpapi's catalogService interface
// documents: it lets handler tests exercise HTTP behavior (status codes,
// JSON shapes, routing, error mapping) against a fake, with no real
// service, repository, or database involved. Unexported for the same
// reason catalogService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type serviceProfileService interface {
	Get(ctx context.Context, id uuid.UUID) (serviceprofile.ServiceProfile, error)
	List(ctx context.Context) ([]serviceprofile.ServiceProfile, error)
	Create(ctx context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error)
	Update(ctx context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ServiceProfileHandler serves the Service Profile REST endpoints:
//
//	POST   /api/v1/service-profiles
//	GET    /api/v1/service-profiles
//	GET    /api/v1/service-profiles/{id}
//	PUT    /api/v1/service-profiles/{id}
//	DELETE /api/v1/service-profiles/{id}
//
// It depends only on serviceProfileService — never a repository directly
// — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is ServiceProfileService's job.
type ServiceProfileHandler struct {
	profiles serviceProfileService
}

// NewServiceProfileHandler builds a ServiceProfileHandler.
func NewServiceProfileHandler(profiles serviceProfileService) *ServiceProfileHandler {
	return &ServiceProfileHandler{profiles: profiles}
}

// Create handles POST /api/v1/service-profiles.
func (h *ServiceProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req serviceProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.profiles.Create(r.Context(), req.toServiceProfile(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newServiceProfileResponse(created))
}

// List handles GET /api/v1/service-profiles.
func (h *ServiceProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.profiles.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceProfileListResponse(profiles))
}

// Get handles GET /api/v1/service-profiles/{id}.
func (h *ServiceProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	p, err := h.profiles.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceProfileResponse(p))
}

// Update handles PUT /api/v1/service-profiles/{id}.
func (h *ServiceProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req serviceProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.profiles.Update(r.Context(), req.toServiceProfile(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceProfileResponse(updated))
}

// Delete handles DELETE /api/v1/service-profiles/{id}.
func (h *ServiceProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.profiles.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
