package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// rackService is the seam RackHandler depends on instead of a concrete
// *service.RackService. See siteService's doc comment for why.
type rackService interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Rack, error)
	List(ctx context.Context) ([]inventory.Rack, error)
	Create(ctx context.Context, rack inventory.Rack) (inventory.Rack, error)
	Update(ctx context.Context, rack inventory.Rack) (inventory.Rack, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// RackHandler serves the Rack REST endpoints:
//
//	POST   /api/v1/racks
//	GET    /api/v1/racks
//	GET    /api/v1/racks/{id}
//	PUT    /api/v1/racks/{id}
//	DELETE /api/v1/racks/{id}
//
// It depends only on rackService — never a repository directly — so it
// has no knowledge of PostgreSQL, SQL, or any storage technology.
type RackHandler struct {
	racks rackService
}

// NewRackHandler builds a RackHandler.
func NewRackHandler(racks rackService) *RackHandler {
	return &RackHandler{racks: racks}
}

// Create handles POST /api/v1/racks.
func (h *RackHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req rackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.racks.Create(r.Context(), req.toRack(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newRackResponse(created))
}

// List handles GET /api/v1/racks.
func (h *RackHandler) List(w http.ResponseWriter, r *http.Request) {
	racks, err := h.racks.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newRackListResponse(racks))
}

// Get handles GET /api/v1/racks/{id}.
func (h *RackHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	rack, err := h.racks.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newRackResponse(rack))
}

// Update handles PUT /api/v1/racks/{id}.
func (h *RackHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req rackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.racks.Update(r.Context(), req.toRack(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newRackResponse(updated))
}

// Delete handles DELETE /api/v1/racks/{id}.
func (h *RackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.racks.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
