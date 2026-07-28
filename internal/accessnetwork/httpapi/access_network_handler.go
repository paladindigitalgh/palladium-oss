package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// accessNetworkService is the seam AccessNetworkHandler depends on
// instead of a concrete *service.AccessNetworkService — the same
// reasoning internal/catalog/httpapi's catalogService interface
// documents: it lets handler tests exercise HTTP behavior (status codes,
// JSON shapes, routing, error mapping) against a fake, with no real
// service, repository, or database involved. Unexported for the same
// reason catalogService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type accessNetworkService interface {
	Get(ctx context.Context, id uuid.UUID) (accessnetwork.AccessNetwork, error)
	List(ctx context.Context) ([]accessnetwork.AccessNetwork, error)
	Create(ctx context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error)
	Update(ctx context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// AccessNetworkHandler serves the Access Network REST endpoints:
//
//	POST   /api/v1/access-networks
//	GET    /api/v1/access-networks
//	GET    /api/v1/access-networks/{id}
//	PUT    /api/v1/access-networks/{id}
//	DELETE /api/v1/access-networks/{id}
//
// It depends only on accessNetworkService — never a repository directly
// — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is AccessNetworkService's job.
type AccessNetworkHandler struct {
	networks accessNetworkService
}

// NewAccessNetworkHandler builds an AccessNetworkHandler.
func NewAccessNetworkHandler(networks accessNetworkService) *AccessNetworkHandler {
	return &AccessNetworkHandler{networks: networks}
}

// Create handles POST /api/v1/access-networks.
func (h *AccessNetworkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req accessNetworkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.networks.Create(r.Context(), req.toAccessNetwork(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newAccessNetworkResponse(created))
}

// List handles GET /api/v1/access-networks.
func (h *AccessNetworkHandler) List(w http.ResponseWriter, r *http.Request) {
	networks, err := h.networks.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessNetworkListResponse(networks))
}

// Get handles GET /api/v1/access-networks/{id}.
func (h *AccessNetworkHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	a, err := h.networks.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessNetworkResponse(a))
}

// Update handles PUT /api/v1/access-networks/{id}.
func (h *AccessNetworkHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req accessNetworkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.networks.Update(r.Context(), req.toAccessNetwork(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessNetworkResponse(updated))
}

// Delete handles DELETE /api/v1/access-networks/{id}.
func (h *AccessNetworkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.networks.Delete(r.Context(), id); err != nil {
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
