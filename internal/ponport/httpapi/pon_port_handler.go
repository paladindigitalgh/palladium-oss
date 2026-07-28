package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// ponPortService is the seam PONPortHandler depends on instead of a
// concrete *service.PONPortService — the same reasoning
// internal/olt/httpapi's oltService interface documents: it lets handler
// tests exercise HTTP behavior (status codes, JSON shapes, routing,
// error mapping) against a fake, with no real service, repository, or
// database involved. Unexported for the same reason oltService is: Go
// interfaces are satisfied structurally, so nothing outside this package
// needs to name it.
type ponPortService interface {
	Get(ctx context.Context, id uuid.UUID) (ponport.PONPort, error)
	List(ctx context.Context) ([]ponport.PONPort, error)
	Create(ctx context.Context, p ponport.PONPort) (ponport.PONPort, error)
	Update(ctx context.Context, p ponport.PONPort) (ponport.PONPort, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PONPortHandler serves the PON Port REST endpoints:
//
//	POST   /api/v1/pon-ports
//	GET    /api/v1/pon-ports
//	GET    /api/v1/pon-ports/{id}
//	PUT    /api/v1/pon-ports/{id}
//	DELETE /api/v1/pon-ports/{id}
//
// It depends only on ponPortService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is PONPortService's job.
type PONPortHandler struct {
	ports ponPortService
}

// NewPONPortHandler builds a PONPortHandler.
func NewPONPortHandler(ports ponPortService) *PONPortHandler {
	return &PONPortHandler{ports: ports}
}

// Create handles POST /api/v1/pon-ports.
func (h *PONPortHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ponPortRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.ports.Create(r.Context(), req.toPONPort(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newPONPortResponse(created))
}

// List handles GET /api/v1/pon-ports.
func (h *PONPortHandler) List(w http.ResponseWriter, r *http.Request) {
	ports, err := h.ports.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newPONPortListResponse(ports))
}

// Get handles GET /api/v1/pon-ports/{id}.
func (h *PONPortHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	p, err := h.ports.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newPONPortResponse(p))
}

// Update handles PUT /api/v1/pon-ports/{id}.
func (h *PONPortHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req ponPortRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.ports.Update(r.Context(), req.toPONPort(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newPONPortResponse(updated))
}

// Delete handles DELETE /api/v1/pon-ports/{id}.
func (h *PONPortHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.ports.Delete(r.Context(), id); err != nil {
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
