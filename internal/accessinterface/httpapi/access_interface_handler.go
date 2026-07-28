package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// accessInterfaceService is the seam AccessInterfaceHandler depends on
// instead of a concrete *service.AccessInterfaceService — the same
// reasoning internal/olt/httpapi's oltService interface documents: it
// lets handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// oltService is: Go interfaces are satisfied structurally, so nothing
// outside this package needs to name it.
type accessInterfaceService interface {
	Get(ctx context.Context, id uuid.UUID) (accessinterface.AccessInterface, error)
	List(ctx context.Context) ([]accessinterface.AccessInterface, error)
	Create(ctx context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error)
	Update(ctx context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// AccessInterfaceHandler serves the Access Interface REST endpoints:
//
//	POST   /api/v1/access-interfaces
//	GET    /api/v1/access-interfaces
//	GET    /api/v1/access-interfaces/{id}
//	PUT    /api/v1/access-interfaces/{id}
//	DELETE /api/v1/access-interfaces/{id}
//
// It depends only on accessInterfaceService — never a repository
// directly — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is AccessInterfaceService's job.
type AccessInterfaceHandler struct {
	interfaces accessInterfaceService
}

// NewAccessInterfaceHandler builds an AccessInterfaceHandler.
func NewAccessInterfaceHandler(interfaces accessInterfaceService) *AccessInterfaceHandler {
	return &AccessInterfaceHandler{interfaces: interfaces}
}

// Create handles POST /api/v1/access-interfaces.
func (h *AccessInterfaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req accessInterfaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.interfaces.Create(r.Context(), req.toAccessInterface(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newAccessInterfaceResponse(created))
}

// List handles GET /api/v1/access-interfaces.
func (h *AccessInterfaceHandler) List(w http.ResponseWriter, r *http.Request) {
	interfaces, err := h.interfaces.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessInterfaceListResponse(interfaces))
}

// Get handles GET /api/v1/access-interfaces/{id}.
func (h *AccessInterfaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	a, err := h.interfaces.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessInterfaceResponse(a))
}

// Update handles PUT /api/v1/access-interfaces/{id}.
func (h *AccessInterfaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req accessInterfaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.interfaces.Update(r.Context(), req.toAccessInterface(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessInterfaceResponse(updated))
}

// Delete handles DELETE /api/v1/access-interfaces/{id}.
func (h *AccessInterfaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.interfaces.Delete(r.Context(), id); err != nil {
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
