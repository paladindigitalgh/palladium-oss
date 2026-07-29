package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// connectionProfileService is the seam ConnectionProfileHandler depends
// on instead of a concrete *service.ConnectionProfileService — the same
// reasoning internal/catalog/httpapi's catalogService interface
// documents: it lets handler tests exercise HTTP behavior (status codes,
// JSON shapes, routing, error mapping) against a fake, with no real
// service, repository, or database involved. Unexported for the same
// reason catalogService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type connectionProfileService interface {
	Get(ctx context.Context, id uuid.UUID) (connectionprofile.ConnectionProfile, error)
	List(ctx context.Context) ([]connectionprofile.ConnectionProfile, error)
	Create(ctx context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error)
	Update(ctx context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ConnectionProfileHandler serves the Connection Profile REST
// endpoints:
//
//	POST   /api/v1/connection-profiles
//	GET    /api/v1/connection-profiles
//	GET    /api/v1/connection-profiles/{id}
//	PUT    /api/v1/connection-profiles/{id}
//	DELETE /api/v1/connection-profiles/{id}
//
// It depends only on connectionProfileService — never a repository
// directly — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is ConnectionProfileService's job.
type ConnectionProfileHandler struct {
	profiles connectionProfileService
}

// NewConnectionProfileHandler builds a ConnectionProfileHandler.
func NewConnectionProfileHandler(profiles connectionProfileService) *ConnectionProfileHandler {
	return &ConnectionProfileHandler{profiles: profiles}
}

// Create handles POST /api/v1/connection-profiles.
func (h *ConnectionProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req connectionProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.profiles.Create(r.Context(), req.toConnectionProfile(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newConnectionProfileResponse(created))
}

// List handles GET /api/v1/connection-profiles.
func (h *ConnectionProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.profiles.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newConnectionProfileListResponse(profiles))
}

// Get handles GET /api/v1/connection-profiles/{id}.
func (h *ConnectionProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	httpx.WriteJSON(w, http.StatusOK, newConnectionProfileResponse(p))
}

// Update handles PUT /api/v1/connection-profiles/{id}.
func (h *ConnectionProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req connectionProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.profiles.Update(r.Context(), req.toConnectionProfile(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newConnectionProfileResponse(updated))
}

// Delete handles DELETE /api/v1/connection-profiles/{id}.
func (h *ConnectionProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
