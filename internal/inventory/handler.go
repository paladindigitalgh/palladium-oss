package inventory

import (
	"encoding/json"
	"net/http"
)

// Handler serves temporary inventory introspection endpoints. It exists
// only to verify the domain is wired into the HTTP router correctly before
// real CRUD endpoints are built, and should be removed once those land.
//
// It has no dependencies today, but is still built via a constructor for
// consistency with the rest of the codebase (see internal/health.Handler).
type Handler struct{}

// NewHandler builds a Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Schema writes the inventory hierarchy as JSON.
func (h *Handler) Schema(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Hierarchy())
}
