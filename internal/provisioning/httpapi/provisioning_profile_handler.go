package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// provisioningProfileService is the seam ProvisioningProfileHandler
// depends on instead of a concrete *service.ProvisioningProfileService —
// the same reasoning internal/product/httpapi's productService interface
// documents: it lets handler tests exercise HTTP behavior (status codes,
// JSON shapes, routing, error mapping) against a fake, with no real
// service, repository, or database involved.
type provisioningProfileService interface {
	Get(ctx context.Context, id uuid.UUID) (provisioning.ProvisioningProfile, error)
	List(ctx context.Context) ([]provisioning.ProvisioningProfile, error)
	Create(ctx context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error)
	Update(ctx context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProvisioningProfileHandler serves the ProvisioningProfile REST
// endpoints:
//
//	POST   /api/v1/provisioning-profiles
//	GET    /api/v1/provisioning-profiles
//	GET    /api/v1/provisioning-profiles/{id}
//	PUT    /api/v1/provisioning-profiles/{id}
//	DELETE /api/v1/provisioning-profiles/{id}
//
// It depends only on provisioningProfileService — never a repository
// directly — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is ProvisioningProfileService's job.
type ProvisioningProfileHandler struct {
	profiles provisioningProfileService
}

// NewProvisioningProfileHandler builds a ProvisioningProfileHandler.
func NewProvisioningProfileHandler(profiles provisioningProfileService) *ProvisioningProfileHandler {
	return &ProvisioningProfileHandler{profiles: profiles}
}

// Create handles POST /api/v1/provisioning-profiles.
func (h *ProvisioningProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req provisioningProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.profiles.Create(r.Context(), req.toProvisioningProfile(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newProvisioningProfileResponse(created))
}

// List handles GET /api/v1/provisioning-profiles.
func (h *ProvisioningProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.profiles.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningProfileListResponse(profiles))
}

// Get handles GET /api/v1/provisioning-profiles/{id}.
func (h *ProvisioningProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	httpx.WriteJSON(w, http.StatusOK, newProvisioningProfileResponse(p))
}

// Update handles PUT /api/v1/provisioning-profiles/{id}.
func (h *ProvisioningProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req provisioningProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.profiles.Update(r.Context(), req.toProvisioningProfile(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProvisioningProfileResponse(updated))
}

// Delete handles DELETE /api/v1/provisioning-profiles/{id}.
func (h *ProvisioningProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
