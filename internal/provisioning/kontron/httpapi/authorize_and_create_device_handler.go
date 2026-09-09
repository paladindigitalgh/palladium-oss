package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// authorizeAndCreateDeviceService is the seam
// AuthorizeAndCreateDeviceHandler depends on instead of a concrete
// *service.AuthorizeAndCreateDeviceService -- the same reasoning
// authorizationService documents in authorization_handler.go.
type authorizeAndCreateDeviceService interface {
	AuthorizeAndCreateDevice(ctx context.Context, oltID uuid.UUID, port string, device inventory.Device) (inventory.Device, string, error)
}

// AuthorizeAndCreateDeviceHandler serves the merged
// authorize-and-create-Device REST endpoint:
//
//	POST /api/v1/provisioning/olts/{oltId}/authorize-and-create-device
//
// This is the UI's entry point for turning one blacklisted ONU into a
// real, tracked Device in a single action -- see
// service.AuthorizeAndCreateDeviceService's own doc comment for why
// AuthorizationHandler's plain authorize-onu endpoint alone was not
// enough. Like AuthorizationHandler, this is POST: it does real,
// non-idempotent work against external hardware as well as Palladium's
// own database.
type AuthorizeAndCreateDeviceHandler struct {
	service authorizeAndCreateDeviceService
}

// NewAuthorizeAndCreateDeviceHandler builds an
// AuthorizeAndCreateDeviceHandler.
func NewAuthorizeAndCreateDeviceHandler(service authorizeAndCreateDeviceService) *AuthorizeAndCreateDeviceHandler {
	return &AuthorizeAndCreateDeviceHandler{service: service}
}

// AuthorizeAndCreateDevice handles
// POST /api/v1/provisioning/olts/{oltId}/authorize-and-create-device.
func (h *AuthorizeAndCreateDeviceHandler) AuthorizeAndCreateDevice(w http.ResponseWriter, r *http.Request) {
	oltID, err := pathOLTID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req authorizeAndCreateDeviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	device, iface, err := h.service.AuthorizeAndCreateDevice(r.Context(), oltID, req.Port, req.toDevice())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newAuthorizeAndCreateDeviceResponse(device, iface))
}
