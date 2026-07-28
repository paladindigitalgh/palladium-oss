package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// oltService is the seam OLTHandler depends on instead of a concrete
// *service.OLTService — the same reasoning
// internal/product/httpapi's productService interface documents: it
// lets handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// productService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type oltService interface {
	Get(ctx context.Context, id uuid.UUID) (olt.OLT, error)
	List(ctx context.Context) ([]olt.OLT, error)
	Create(ctx context.Context, o olt.OLT) (olt.OLT, error)
	Update(ctx context.Context, o olt.OLT) (olt.OLT, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// OLTHandler serves the OLT REST endpoints:
//
//	POST   /api/v1/olts
//	GET    /api/v1/olts
//	GET    /api/v1/olts/{id}
//	PUT    /api/v1/olts/{id}
//	DELETE /api/v1/olts/{id}
//
// It depends only on oltService — never a repository directly — so it
// has no knowledge of PostgreSQL, SQL, or any storage technology. Every
// method is a thin decode/delegate/translate, with no business logic:
// that is OLTService's job.
type OLTHandler struct {
	olts oltService
}

// NewOLTHandler builds an OLTHandler.
func NewOLTHandler(olts oltService) *OLTHandler {
	return &OLTHandler{olts: olts}
}

// Create handles POST /api/v1/olts.
func (h *OLTHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req oltRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.olts.Create(r.Context(), req.toOLT(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newOLTResponse(created))
}

// List handles GET /api/v1/olts.
func (h *OLTHandler) List(w http.ResponseWriter, r *http.Request) {
	olts, err := h.olts.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newOLTListResponse(olts))
}

// Get handles GET /api/v1/olts/{id}.
func (h *OLTHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	o, err := h.olts.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newOLTResponse(o))
}

// Update handles PUT /api/v1/olts/{id}.
func (h *OLTHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req oltRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.olts.Update(r.Context(), req.toOLT(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newOLTResponse(updated))
}

// Delete handles DELETE /api/v1/olts/{id}.
func (h *OLTHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.olts.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
