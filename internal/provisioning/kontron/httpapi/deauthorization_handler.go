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
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// deauthorizationService is the seam DeauthorizationHandler depends on
// instead of a concrete *service.DeauthorizationService — the same
// reasoning authorizationService's own doc comment documents.
type deauthorizationService interface {
	DeauthorizeONU(ctx context.Context, deviceID uuid.UUID) (string, error)
}

// deviceGetter is the seam DeauthorizationHandler uses to resolve the
// retired Device's own Name for its Event message — the minimal slice
// of inventory.DeviceRepository it actually needs, the same "depend on
// the seam, not the concrete type" reasoning deauthorizationService
// above already follows.
type deviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
}

// eventRecorder is the seam DeauthorizationHandler uses to write an
// operational Event after a successful deauthorization — see
// internal/inventory/httpapi.DeviceHandler's own eventRecorder for why
// this lives in the handler, not service.DeauthorizationService.
type eventRecorder interface {
	Create(ctx context.Context, e event.Event) (event.Event, error)
}

// DeauthorizationHandler serves the "Delete ONU" REST endpoint:
//
//	POST /api/v1/provisioning/devices/{deviceId}/deauthorize-onu
//
// Unlike AuthorizationHandler, this is keyed by deviceId, not oltId:
// this is triggered from the Device Detail page, which has a DeviceID in
// hand and nothing else, so service.DeauthorizationService resolves the
// OLT and interface itself rather than requiring the caller to already
// know them. Like AuthorizeONU, this is POST: it runs real,
// non-idempotent work against external hardware, more so even than a
// diagnostic read.
type DeauthorizationHandler struct {
	deauthorization deauthorizationService
	devices         deviceGetter
	events          eventRecorder
}

// NewDeauthorizationHandler builds a DeauthorizationHandler.
func NewDeauthorizationHandler(deauthorization deauthorizationService, devices deviceGetter, events eventRecorder) *DeauthorizationHandler {
	return &DeauthorizationHandler{deauthorization: deauthorization, devices: devices, events: events}
}

// DeauthorizeONU handles POST /api/v1/provisioning/devices/{deviceId}/deauthorize-onu.
// service.DeauthorizationService.DeauthorizeONU always retires the
// Device on success (see that method's own doc comment), so this always
// records a "device.retired" Event, not a conditional one.
func (h *DeauthorizationHandler) DeauthorizeONU(w http.ResponseWriter, r *http.Request) {
	deviceID, err := pathDeviceID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	iface, err := h.deauthorization.DeauthorizeONU(r.Context(), deviceID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	device, err := h.devices.Get(r.Context(), deviceID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	if _, err := h.events.Create(r.Context(), event.Event{
		EntityType:  "device",
		EntityID:    device.ID,
		Type:        "device.retired",
		Message:     fmt.Sprintf("Retired device %s", device.Name),
		ActorUserID: actorUserID,
	}); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, authorizeONUResponse{Interface: iface})
}

func pathDeviceID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "deviceId"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("deviceId must be a valid UUID")
	}
	return id, nil
}
