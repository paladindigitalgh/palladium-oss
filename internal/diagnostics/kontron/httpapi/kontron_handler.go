package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// kontronService is the seam KontronHandler depends on instead of a
// concrete *service.KontronService — the same reasoning every other
// domain's *service interface in this codebase documents: it lets
// handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service, Dialer,
// or SSH connection involved.
type kontronService interface {
	ONUSummary(ctx context.Context, oltID uuid.UUID) (string, error)
	ONUStatusSummary(ctx context.Context, oltID uuid.UUID) (string, error)
	ONURunningConfig(ctx context.Context, oltID uuid.UUID, iface string) (string, error)
	ONUDetail(ctx context.Context, oltID uuid.UUID, iface string) (string, error)
	ONUStatus(ctx context.Context, oltID uuid.UUID, iface string) (string, error)
	ONUEthernetPorts(ctx context.Context, oltID uuid.UUID, iface string) (string, error)
	DHCPSnoopingEntries(ctx context.Context, oltID uuid.UUID, iface string) (string, error)
	MACAddressTableEntries(ctx context.Context, oltID uuid.UUID, iface string) (string, error)
}

// KontronHandler serves the Kontron diagnostics REST endpoints, every
// one scoped to a specific OLT by path parameter:
//
//	POST /api/v1/diagnostics/olts/{oltId}/onu-summary
//	POST /api/v1/diagnostics/olts/{oltId}/onu-status-summary
//	POST /api/v1/diagnostics/olts/{oltId}/onu-running-config
//	POST /api/v1/diagnostics/olts/{oltId}/onu-detail
//	POST /api/v1/diagnostics/olts/{oltId}/onu-status
//	POST /api/v1/diagnostics/olts/{oltId}/onu-ethernet-ports
//	POST /api/v1/diagnostics/olts/{oltId}/dhcp-snooping-entries
//	POST /api/v1/diagnostics/olts/{oltId}/mac-address-table-entries
//
// Every method is a thin decode/delegate/translate, with no business
// logic: that is KontronService's job. All eight are POST, matching
// internal/diagnostics/httpapi.DiagnosticsHandler.BasicONUCheck's own
// precedent — each one does real, non-idempotent work against external
// hardware (opening an SSH connection, running a command), not a cached
// or side-effect-free resource fetch a GET would imply.
type KontronHandler struct {
	kontron kontronService
}

// NewKontronHandler builds a KontronHandler.
func NewKontronHandler(kontron kontronService) *KontronHandler {
	return &KontronHandler{kontron: kontron}
}

// ONUSummary handles POST /api/v1/diagnostics/olts/{oltId}/onu-summary.
func (h *KontronHandler) ONUSummary(w http.ResponseWriter, r *http.Request) {
	h.runNoArgs(w, r, h.kontron.ONUSummary)
}

// ONUStatusSummary handles POST
// /api/v1/diagnostics/olts/{oltId}/onu-status-summary.
func (h *KontronHandler) ONUStatusSummary(w http.ResponseWriter, r *http.Request) {
	h.runNoArgs(w, r, h.kontron.ONUStatusSummary)
}

// ONURunningConfig handles POST
// /api/v1/diagnostics/olts/{oltId}/onu-running-config.
func (h *KontronHandler) ONURunningConfig(w http.ResponseWriter, r *http.Request) {
	h.runForInterface(w, r, h.kontron.ONURunningConfig)
}

// ONUDetail handles POST /api/v1/diagnostics/olts/{oltId}/onu-detail.
func (h *KontronHandler) ONUDetail(w http.ResponseWriter, r *http.Request) {
	h.runForInterface(w, r, h.kontron.ONUDetail)
}

// ONUStatus handles POST /api/v1/diagnostics/olts/{oltId}/onu-status.
func (h *KontronHandler) ONUStatus(w http.ResponseWriter, r *http.Request) {
	h.runForInterface(w, r, h.kontron.ONUStatus)
}

// ONUEthernetPorts handles POST
// /api/v1/diagnostics/olts/{oltId}/onu-ethernet-ports.
func (h *KontronHandler) ONUEthernetPorts(w http.ResponseWriter, r *http.Request) {
	h.runForInterface(w, r, h.kontron.ONUEthernetPorts)
}

// DHCPSnoopingEntries handles POST
// /api/v1/diagnostics/olts/{oltId}/dhcp-snooping-entries.
func (h *KontronHandler) DHCPSnoopingEntries(w http.ResponseWriter, r *http.Request) {
	h.runForInterface(w, r, h.kontron.DHCPSnoopingEntries)
}

// MACAddressTableEntries handles POST
// /api/v1/diagnostics/olts/{oltId}/mac-address-table-entries.
func (h *KontronHandler) MACAddressTableEntries(w http.ResponseWriter, r *http.Request) {
	h.runForInterface(w, r, h.kontron.MACAddressTableEntries)
}

// runNoArgs is the shared decode/delegate/respond sequence for the two
// whole-OLT endpoints (ONUSummary, ONUStatusSummary), which take no
// request body.
func (h *KontronHandler) runNoArgs(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, oltID uuid.UUID) (string, error)) {
	oltID, err := pathOLTID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	out, err := fn(r.Context(), oltID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, commandOutputResponse{Output: out})
}

// runForInterface is the shared decode/validate/delegate/respond
// sequence for every per-interface endpoint (everything but ONUSummary
// and ONUStatusSummary).
func (h *KontronHandler) runForInterface(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, oltID uuid.UUID, iface string) (string, error)) {
	oltID, err := pathOLTID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req interfaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	out, err := fn(r.Context(), oltID, req.Interface)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, commandOutputResponse{Output: out})
}

func pathOLTID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "oltId"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("oltId must be a valid UUID")
	}
	return id, nil
}
