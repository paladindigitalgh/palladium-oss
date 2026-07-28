package httpapi

import (
	"context"
	"net/http"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
)

// diagnosticsService is the seam DiagnosticsHandler depends on instead
// of a concrete *service.DiagnosticsService — the same reasoning every
// other domain's *Service interface in this codebase documents: it lets
// handler tests exercise HTTP behavior (status codes, JSON shapes, error
// mapping) against a fake, with no real service or Registry involved.
// Unexported for the same reason every other domain's is: Go interfaces
// are satisfied structurally, so nothing outside this package needs to
// name it.
type diagnosticsService interface {
	Run(ctx context.Context, name string, request diagnostics.Request) (*diagnostics.Result, error)
}

// DiagnosticsHandler serves the Diagnostics framework's REST endpoint:
//
//	POST /api/v1/diagnostics/basic-onu-check
//
// It depends only on diagnosticsService — never a Registry directly —
// so it has no knowledge of how a diagnostic is looked up or executed.
// BasicONUCheck is a thin decode/delegate/translate method, with no
// business logic: that is DiagnosticsService's (and, beneath it,
// diagnostics.BasicONUCheck's) job.
//
// This handler names diagnostics.BasicONUCheckName directly rather than
// accepting a diagnostic name as a path or query parameter: this
// milestone defines exactly one built-in diagnostic and exactly one
// fixed route for it (goal 7), not a generic "run any registered
// diagnostic by name" endpoint — that generality is not asked for, and
// CLAUDE.md's general rule favors the simpler option absent a concrete
// need for more.
type DiagnosticsHandler struct {
	diagnostics diagnosticsService
}

// NewDiagnosticsHandler builds a DiagnosticsHandler.
func NewDiagnosticsHandler(diagnosticsSvc diagnosticsService) *DiagnosticsHandler {
	return &DiagnosticsHandler{diagnostics: diagnosticsSvc}
}

// BasicONUCheck handles POST /api/v1/diagnostics/basic-onu-check.
func (h *DiagnosticsHandler) BasicONUCheck(w http.ResponseWriter, r *http.Request) {
	var req basicONUCheckRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	result, err := h.diagnostics.Run(r.Context(), diagnostics.BasicONUCheckName, req.toRequest())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newResultResponse(*result))
}
