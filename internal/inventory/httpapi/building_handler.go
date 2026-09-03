package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// buildingService is the seam BuildingHandler depends on instead of a
// concrete *service.BuildingService. See siteService's doc comment for
// why.
type buildingService interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Building, error)
	List(ctx context.Context) ([]inventory.Building, error)
	Create(ctx context.Context, building inventory.Building) (inventory.Building, error)
	Update(ctx context.Context, building inventory.Building) (inventory.Building, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// BuildingHandler serves the Building REST endpoints:
//
//	POST   /api/v1/buildings
//	GET    /api/v1/buildings
//	GET    /api/v1/buildings/{id}
//	PUT    /api/v1/buildings/{id}
//	DELETE /api/v1/buildings/{id}
//
// It depends only on buildingService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
type BuildingHandler struct {
	buildings buildingService
}

// NewBuildingHandler builds a BuildingHandler.
func NewBuildingHandler(buildings buildingService) *BuildingHandler {
	return &BuildingHandler{buildings: buildings}
}

// Create handles POST /api/v1/buildings.
func (h *BuildingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req buildingRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.buildings.Create(r.Context(), req.toBuilding(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newBuildingResponse(created))
}

// List handles GET /api/v1/buildings.
func (h *BuildingHandler) List(w http.ResponseWriter, r *http.Request) {
	buildings, err := h.buildings.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newBuildingListResponse(buildings))
}

// Get handles GET /api/v1/buildings/{id}.
func (h *BuildingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	building, err := h.buildings.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newBuildingResponse(building))
}

// Update handles PUT /api/v1/buildings/{id}.
func (h *BuildingHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req buildingRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.buildings.Update(r.Context(), req.toBuilding(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newBuildingResponse(updated))
}

// Delete handles DELETE /api/v1/buildings/{id}.
func (h *BuildingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.buildings.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
