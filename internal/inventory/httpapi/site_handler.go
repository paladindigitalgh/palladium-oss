package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// siteService is the seam SiteHandler depends on instead of a concrete
// *service.SiteService. This is the same reasoning that has
// SiteRepository depend on database.Querier instead of *database.Pool one
// layer down (internal/inventory/postgres/site.go): it lets handler tests
// exercise HTTP behavior — status codes, JSON shapes, routing, error
// mapping — against a fake, with no real service, repository, or
// database involved. Unexported: nothing outside this package needs to
// name it, since Go interfaces are satisfied structurally — a caller in
// another package can pass *service.SiteService into NewSiteHandler
// without ever referring to this type.
type siteService interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Site, error)
	List(ctx context.Context) ([]inventory.Site, error)
	Create(ctx context.Context, site inventory.Site) (inventory.Site, error)
	Update(ctx context.Context, site inventory.Site) (inventory.Site, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// SiteHandler serves the Site REST endpoints:
//
//	POST   /api/v1/sites
//	GET    /api/v1/sites
//	GET    /api/v1/sites/{id}
//	PUT    /api/v1/sites/{id}
//	DELETE /api/v1/sites/{id}
//
// It depends only on siteService — never a repository directly (goal 1) —
// so it has no knowledge of PostgreSQL, SQL, or any storage technology.
type SiteHandler struct {
	sites siteService
}

// NewSiteHandler builds a SiteHandler.
func NewSiteHandler(sites siteService) *SiteHandler {
	return &SiteHandler{sites: sites}
}

// Create handles POST /api/v1/sites.
func (h *SiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req siteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.sites.Create(r.Context(), req.toSite(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newSiteResponse(created))
}

// List handles GET /api/v1/sites.
func (h *SiteHandler) List(w http.ResponseWriter, r *http.Request) {
	sites, err := h.sites.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newSiteListResponse(sites))
}

// Get handles GET /api/v1/sites/{id}.
func (h *SiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	site, err := h.sites.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newSiteResponse(site))
}

// Update handles PUT /api/v1/sites/{id}.
func (h *SiteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req siteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.sites.Update(r.Context(), req.toSite(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newSiteResponse(updated))
}

// Delete handles DELETE /api/v1/sites/{id}.
func (h *SiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.sites.Delete(r.Context(), id); err != nil {
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
