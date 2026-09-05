package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
)

// providerService is the seam ProviderHandler depends on instead of a
// concrete *service.ProviderService — the same reasoning
// internal/serviceprofile/httpapi's serviceProfileService interface
// documents: it lets handler tests exercise HTTP behavior (status
// codes, JSON shapes, routing, error mapping) against a fake, with no
// real service, repository, or database involved.
type providerService interface {
	Get(ctx context.Context, id uuid.UUID) (provider.Provider, error)
	List(ctx context.Context) ([]provider.Provider, error)
	Create(ctx context.Context, p provider.Provider) (provider.Provider, error)
	Update(ctx context.Context, p provider.Provider) (provider.Provider, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProviderHandler serves the Provider REST endpoints:
//
//	POST   /api/v1/providers
//	GET    /api/v1/providers
//	GET    /api/v1/providers/{id}
//	PUT    /api/v1/providers/{id}
//	DELETE /api/v1/providers/{id}
//
// It depends only on providerService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is ProviderService's job.
type ProviderHandler struct {
	providers providerService
}

// NewProviderHandler builds a ProviderHandler.
func NewProviderHandler(providers providerService) *ProviderHandler {
	return &ProviderHandler{providers: providers}
}

// Create handles POST /api/v1/providers.
func (h *ProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req providerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.providers.Create(r.Context(), req.toProvider(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newProviderResponse(created))
}

// List handles GET /api/v1/providers.
func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	providers, err := h.providers.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProviderListResponse(providers))
}

// Get handles GET /api/v1/providers/{id}.
func (h *ProviderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	p, err := h.providers.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProviderResponse(p))
}

// Update handles PUT /api/v1/providers/{id}.
func (h *ProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req providerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.providers.Update(r.Context(), req.toProvider(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProviderResponse(updated))
}

// Delete handles DELETE /api/v1/providers/{id}.
func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.providers.Delete(r.Context(), id); err != nil {
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
