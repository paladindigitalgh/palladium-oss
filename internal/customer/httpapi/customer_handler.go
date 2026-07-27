package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// customerService is the seam CustomerHandler depends on instead of a
// concrete *service.CustomerService — the same reasoning
// internal/inventory/httpapi's siteService interface documents: it lets
// handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// siteService is: Go interfaces are satisfied structurally, so nothing
// outside this package needs to name it.
type customerService interface {
	Get(ctx context.Context, id uuid.UUID) (customer.Customer, error)
	List(ctx context.Context) ([]customer.Customer, error)
	Create(ctx context.Context, c customer.Customer) (customer.Customer, error)
	Update(ctx context.Context, c customer.Customer) (customer.Customer, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CustomerHandler serves the Customer REST endpoints:
//
//	POST   /api/v1/customers
//	GET    /api/v1/customers
//	GET    /api/v1/customers/{id}
//	PUT    /api/v1/customers/{id}
//	DELETE /api/v1/customers/{id}
//
// It depends only on customerService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is CustomerService's job (goal 4).
type CustomerHandler struct {
	customers customerService
}

// NewCustomerHandler builds a CustomerHandler.
func NewCustomerHandler(customers customerService) *CustomerHandler {
	return &CustomerHandler{customers: customers}
}

// Create handles POST /api/v1/customers.
func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.customers.Create(r.Context(), req.toCustomer(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newCustomerResponse(created))
}

// List handles GET /api/v1/customers.
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	customers, err := h.customers.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCustomerListResponse(customers))
}

// Get handles GET /api/v1/customers/{id}.
func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	c, err := h.customers.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCustomerResponse(c))
}

// Update handles PUT /api/v1/customers/{id}.
func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req customerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.customers.Update(r.Context(), req.toCustomer(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCustomerResponse(updated))
}

// Delete handles DELETE /api/v1/customers/{id}.
func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.customers.Delete(r.Context(), id); err != nil {
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
