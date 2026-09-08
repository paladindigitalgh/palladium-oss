package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// deauthorizationService is the seam DeauthorizationHandler depends on
// instead of a concrete *service.DeauthorizationService — the same
// reasoning authorizationService's own doc comment documents.
type deauthorizationService interface {
	DeauthorizeONU(ctx context.Context, deviceID uuid.UUID) (string, error)
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
}

// NewDeauthorizationHandler builds a DeauthorizationHandler.
func NewDeauthorizationHandler(deauthorization deauthorizationService) *DeauthorizationHandler {
	return &DeauthorizationHandler{deauthorization: deauthorization}
}

// DeauthorizeONU handles POST /api/v1/provisioning/devices/{deviceId}/deauthorize-onu.
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

	httpx.WriteJSON(w, http.StatusOK, authorizeONUResponse{Interface: iface})
}

func pathDeviceID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "deviceId"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("deviceId must be a valid UUID")
	}
	return id, nil
}
