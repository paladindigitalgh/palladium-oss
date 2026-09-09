package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// deviceManufacturerService is the seam DeviceManufacturerHandler
// depends on instead of a concrete *service.DeviceManufacturerService —
// the same reasoning internal/oltmodel/httpapi's oltModelService
// interface documents: it lets handler tests exercise HTTP behavior
// (status codes, JSON shapes, routing, error mapping) against a fake,
// with no real service, repository, or database involved.
type deviceManufacturerService interface {
	Get(ctx context.Context, id uuid.UUID) (devicemanufacturer.DeviceManufacturer, error)
	List(ctx context.Context) ([]devicemanufacturer.DeviceManufacturer, error)
	Create(ctx context.Context, m devicemanufacturer.DeviceManufacturer) (devicemanufacturer.DeviceManufacturer, error)
	Update(ctx context.Context, m devicemanufacturer.DeviceManufacturer) (devicemanufacturer.DeviceManufacturer, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// DeviceManufacturerHandler serves the Device Manufacturer REST
// endpoints:
//
//	POST   /api/v1/device-manufacturers
//	GET    /api/v1/device-manufacturers
//	GET    /api/v1/device-manufacturers/{id}
//	PUT    /api/v1/device-manufacturers/{id}
//	DELETE /api/v1/device-manufacturers/{id}
//
// It depends only on deviceManufacturerService — never a repository
// directly — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is DeviceManufacturerService's job.
type DeviceManufacturerHandler struct {
	manufacturers deviceManufacturerService
}

// NewDeviceManufacturerHandler builds a DeviceManufacturerHandler.
func NewDeviceManufacturerHandler(manufacturers deviceManufacturerService) *DeviceManufacturerHandler {
	return &DeviceManufacturerHandler{manufacturers: manufacturers}
}

// Create handles POST /api/v1/device-manufacturers.
func (h *DeviceManufacturerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req deviceManufacturerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.manufacturers.Create(r.Context(), req.toDeviceManufacturer(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newDeviceManufacturerResponse(created))
}

// List handles GET /api/v1/device-manufacturers.
func (h *DeviceManufacturerHandler) List(w http.ResponseWriter, r *http.Request) {
	manufacturers, err := h.manufacturers.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceManufacturerListResponse(manufacturers))
}

// Get handles GET /api/v1/device-manufacturers/{id}.
func (h *DeviceManufacturerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	m, err := h.manufacturers.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceManufacturerResponse(m))
}

// Update handles PUT /api/v1/device-manufacturers/{id}.
func (h *DeviceManufacturerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req deviceManufacturerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.manufacturers.Update(r.Context(), req.toDeviceManufacturer(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceManufacturerResponse(updated))
}

// Delete handles DELETE /api/v1/device-manufacturers/{id}.
func (h *DeviceManufacturerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.manufacturers.Delete(r.Context(), id); err != nil {
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
