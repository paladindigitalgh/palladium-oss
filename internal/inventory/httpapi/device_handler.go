package httpapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// deviceService is the seam DeviceHandler depends on instead of a
// concrete *service.DeviceService. See siteService's doc comment
// (site_handler.go) for why this is unexported and structural.
type deviceService interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
	GetBySerialNumber(ctx context.Context, serialNumber string) (inventory.Device, error)
	List(ctx context.Context) ([]inventory.Device, error)
	Create(ctx context.Context, device inventory.Device) (inventory.Device, error)
	Update(ctx context.Context, device inventory.Device) (inventory.Device, error)
}

// eventRecorder is the seam DeviceHandler uses to write an operational
// Event after a successful Create — the minimal slice of
// event.EventRepository it actually needs, the same "depend on the
// seam, not the concrete type" reasoning deviceService above already
// follows. Event-writing happens here, in the handler, not inside
// DeviceService: see internal/note/httpapi.NoteHandler's own
// userLookup for the established precedent of resolving
// display-friendly context at the handler layer rather than growing a
// domain service's own dependencies for it.
type eventRecorder interface {
	Create(ctx context.Context, e event.Event) (event.Event, error)
}

const entityTypeDevice = "device"

// DeviceHandler serves the Device REST endpoints:
//
//	POST   /api/v1/devices
//	GET    /api/v1/devices
//	GET    /api/v1/devices/{id}
//	GET    /api/v1/devices/by-serial-number/{serialNumber}
//	PUT    /api/v1/devices/{id}
//
// Deliberately no DELETE -- see inventory.DeviceRepository's own doc
// comment for why Palladium never permanently erases a Device.
//
// It depends only on deviceService — never a repository directly — so it
// has no knowledge of PostgreSQL, SQL, or any storage technology.
type DeviceHandler struct {
	devices deviceService
	events  eventRecorder
}

// NewDeviceHandler builds a DeviceHandler.
func NewDeviceHandler(devices deviceService, events eventRecorder) *DeviceHandler {
	return &DeviceHandler{devices: devices, events: events}
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

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	if _, err := h.events.Create(r.Context(), event.Event{
		EntityType:  entityTypeDevice,
		EntityID:    created.ID,
		Type:        "device.created",
		Message:     fmt.Sprintf("Created device %s", created.Name),
		ActorUserID: actorUserID,
	}); err != nil {
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

// GetBySerialNumber handles GET
// /api/v1/devices/by-serial-number/{serialNumber}. A 404 here is the
// normal "no Device with this serial number yet" case a caller (e.g. the
// Discover ONU picker, checking before creating one) is expected to
// handle, not an exceptional error.
func (h *DeviceHandler) GetBySerialNumber(w http.ResponseWriter, r *http.Request) {
	serialNumber := chi.URLParam(r, "serialNumber")

	device, err := h.devices.GetBySerialNumber(r.Context(), serialNumber)
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
