package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer/removal"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
)

// removalService is the seam RemovalHandler depends on instead of a
// concrete *removal.RemovalService — the same reasoning customerService
// documents.
type removalService interface {
	Preview(ctx context.Context, customerID uuid.UUID) (removal.Preview, error)
	Execute(ctx context.Context, customerID uuid.UUID) error
}

// RemovalHandler serves "Remove Customer" (see internal/customer/removal's
// own package doc comment for what that means):
//
//	GET  /api/v1/customers/{id}/removal-preview
//	POST /api/v1/customers/{id}/removal
//
// This is deliberately not folded into CustomerHandler.Delete: that
// endpoint stays a strict, FK-blocked delete for direct/API use, exactly
// as it is today. This is a separate, explicit action a caller opts into.
type RemovalHandler struct {
	removals removalService
}

// NewRemovalHandler builds a RemovalHandler.
func NewRemovalHandler(removals removalService) *RemovalHandler {
	return &RemovalHandler{removals: removals}
}

// Preview handles GET /api/v1/customers/{id}/removal-preview.
func (h *RemovalHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	preview, err := h.removals.Preview(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newRemovalPreviewResponse(preview))
}

// Execute handles POST /api/v1/customers/{id}/removal.
func (h *RemovalHandler) Execute(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.removals.Execute(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
