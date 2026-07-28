package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
)

// serviceService is the seam ServiceHandler depends on instead of a
// concrete *service.ServiceService — the same reasoning
// internal/product/httpapi's productService interface documents: it lets
// handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// productService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type serviceService interface {
	Get(ctx context.Context, id uuid.UUID) (domainservice.Service, error)
	List(ctx context.Context) ([]domainservice.Service, error)
	Create(ctx context.Context, s domainservice.Service) (domainservice.Service, error)
	Update(ctx context.Context, s domainservice.Service) (domainservice.Service, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ServiceHandler serves the Service REST endpoints:
//
//	POST   /api/v1/services
//	GET    /api/v1/services
//	GET    /api/v1/services/{id}
//	PUT    /api/v1/services/{id}
//	DELETE /api/v1/services/{id}
//
// It depends only on serviceService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is ServiceService's job.
type ServiceHandler struct {
	services serviceService
}

// NewServiceHandler builds a ServiceHandler.
func NewServiceHandler(services serviceService) *ServiceHandler {
	return &ServiceHandler{services: services}
}

// Create handles POST /api/v1/services.
func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.services.Create(r.Context(), req.toService(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newServiceResponse(created))
}

// List handles GET /api/v1/services.
func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.services.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceListResponse(services))
}

// Get handles GET /api/v1/services/{id}.
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	s, err := h.services.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceResponse(s))
}

// Update handles PUT /api/v1/services/{id}.
func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req serviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.services.Update(r.Context(), req.toService(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceResponse(updated))
}

// Delete handles DELETE /api/v1/services/{id}.
func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.services.Delete(r.Context(), id); err != nil {
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
