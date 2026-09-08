package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// oltModelService is the seam OLTModelHandler depends on instead of a
// concrete *service.OLTModelService — the same reasoning
// internal/provider/httpapi's providerService interface documents: it
// lets handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved.
type oltModelService interface {
	Get(ctx context.Context, id uuid.UUID) (oltmodel.OLTModel, error)
	List(ctx context.Context) ([]oltmodel.OLTModel, error)
	Create(ctx context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error)
	Update(ctx context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// OLTModelHandler serves the OLT Model REST endpoints:
//
//	POST   /api/v1/olt-models
//	GET    /api/v1/olt-models
//	GET    /api/v1/olt-models/{id}
//	PUT    /api/v1/olt-models/{id}
//	DELETE /api/v1/olt-models/{id}
//
// It depends only on oltModelService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is OLTModelService's job.
type OLTModelHandler struct {
	models oltModelService
}

// NewOLTModelHandler builds an OLTModelHandler.
func NewOLTModelHandler(models oltModelService) *OLTModelHandler {
	return &OLTModelHandler{models: models}
}

// Create handles POST /api/v1/olt-models.
func (h *OLTModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req oltModelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.models.Create(r.Context(), req.toOLTModel(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newOLTModelResponse(created))
}

// List handles GET /api/v1/olt-models.
func (h *OLTModelHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.models.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newOLTModelListResponse(models))
}

// Get handles GET /api/v1/olt-models/{id}.
func (h *OLTModelHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	m, err := h.models.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newOLTModelResponse(m))
}

// Update handles PUT /api/v1/olt-models/{id}.
func (h *OLTModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req oltModelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.models.Update(r.Context(), req.toOLTModel(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newOLTModelResponse(updated))
}

// Delete handles DELETE /api/v1/olt-models/{id}.
func (h *OLTModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.models.Delete(r.Context(), id); err != nil {
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
