package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// catalogService is the seam CatalogHandler depends on instead of a
// concrete *service.CatalogService — the same reasoning
// internal/location/httpapi's locationService interface documents: it
// lets handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// locationService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type catalogService interface {
	Get(ctx context.Context, id uuid.UUID) (catalog.ProductCatalog, error)
	List(ctx context.Context) ([]catalog.ProductCatalog, error)
	Create(ctx context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error)
	Update(ctx context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CatalogHandler serves the Catalog REST endpoints:
//
//	POST   /api/v1/catalogs
//	GET    /api/v1/catalogs
//	GET    /api/v1/catalogs/{id}
//	PUT    /api/v1/catalogs/{id}
//	DELETE /api/v1/catalogs/{id}
//
// It depends only on catalogService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is CatalogService's job.
type CatalogHandler struct {
	catalogs catalogService
}

// NewCatalogHandler builds a CatalogHandler.
func NewCatalogHandler(catalogs catalogService) *CatalogHandler {
	return &CatalogHandler{catalogs: catalogs}
}

// Create handles POST /api/v1/catalogs.
func (h *CatalogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req catalogRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.catalogs.Create(r.Context(), req.toCatalog(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newCatalogResponse(created))
}

// List handles GET /api/v1/catalogs.
func (h *CatalogHandler) List(w http.ResponseWriter, r *http.Request) {
	catalogs, err := h.catalogs.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCatalogListResponse(catalogs))
}

// Get handles GET /api/v1/catalogs/{id}.
func (h *CatalogHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	c, err := h.catalogs.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCatalogResponse(c))
}

// Update handles PUT /api/v1/catalogs/{id}.
func (h *CatalogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req catalogRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.catalogs.Update(r.Context(), req.toCatalog(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCatalogResponse(updated))
}

// Delete handles DELETE /api/v1/catalogs/{id}.
func (h *CatalogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.catalogs.Delete(r.Context(), id); err != nil {
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
