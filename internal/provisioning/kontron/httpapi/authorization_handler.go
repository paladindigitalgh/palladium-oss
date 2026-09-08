package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// authorizationService is the seam AuthorizationHandler depends on
// instead of a concrete *service.AuthorizationService — the same
// reasoning internal/diagnostics/kontron/httpapi's own kontronService
// interface documents: it lets handler tests exercise HTTP behavior
// against a fake, with no real service, Dialer, or SSH connection
// involved.
type authorizationService interface {
	AuthorizeONU(ctx context.Context, oltID uuid.UUID, port, serialNumber string) (string, error)
}

// AuthorizationHandler serves the Kontron ONU-authorization REST
// endpoint:
//
//	POST /api/v1/provisioning/olts/{oltId}/authorize-onu
//
// This is a thin decode/validate/delegate/translate, with no business
// logic: that is service.AuthorizationService's job. It is POST, the
// same reasoning every Kontron diagnostics endpoint already gives (see
// internal/diagnostics/kontron/httpapi's own doc comment): this does
// real, non-idempotent work against external hardware — opening an SSH
// connection and changing the device's actual configuration, more so
// even than a read-only diagnostic command.
type AuthorizationHandler struct {
	authorization authorizationService
}

// NewAuthorizationHandler builds an AuthorizationHandler.
func NewAuthorizationHandler(authorization authorizationService) *AuthorizationHandler {
	return &AuthorizationHandler{authorization: authorization}
}

// AuthorizeONU handles POST /api/v1/provisioning/olts/{oltId}/authorize-onu.
func (h *AuthorizationHandler) AuthorizeONU(w http.ResponseWriter, r *http.Request) {
	oltID, err := pathOLTID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req authorizeONURequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	iface, err := h.authorization.AuthorizeONU(r.Context(), oltID, req.Port, req.SerialNumber)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, authorizeONUResponse{Interface: iface})
}

func pathOLTID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "oltId"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("oltId must be a valid UUID")
	}
	return id, nil
}
