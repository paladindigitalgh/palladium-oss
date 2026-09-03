package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// deviceService is the seam DeviceHandler depends on instead of a
// concrete *service.DeviceService. See siteService's doc comment
// (site_handler.go) for why this is unexported and structural.
type deviceService interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
	List(ctx context.Context) ([]inventory.Device, error)
	Create(ctx context.Context, device inventory.Device) (inventory.Device, error)
	Update(ctx context.Context, device inventory.Device) (inventory.Device, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// DeviceHandler serves the Device REST endpoints:
//
//	POST   /api/v1/devices
//	GET    /api/v1/devices
//	GET    /api/v1/devices/{id}
//	PUT    /api/v1/devices/{id}
//	DELETE /api/v1/devices/{id}
//
// It depends only on deviceService — never a repository directly — so it
// has no knowledge of PostgreSQL, SQL, or any storage technology.
type DeviceHandler struct {
	devices deviceService
}

// NewDeviceHandler builds a DeviceHandler.
func NewDeviceHandler(devices deviceService) *DeviceHandler {
	return &DeviceHandler{devices: devices}
}

// Create handles POST /api/v1/devices.
func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req deviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.devices.Create(r.Context(), req.toDevice(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newDeviceResponse(created))
}

// List handles GET /api/v1/devices.
func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	devices, err := h.devices.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceListResponse(devices))
}

// Get handles GET /api/v1/devices/{id}.
func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	device, err := h.devices.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceResponse(device))
}

// Update handles PUT /api/v1/devices/{id}.
func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req deviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.devices.Update(r.Context(), req.toDevice(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceResponse(updated))
}

// Delete handles DELETE /api/v1/devices/{id}.
func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.devices.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
