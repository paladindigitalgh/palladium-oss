package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

// productService is the seam ProductHandler depends on instead of a
// concrete *service.ProductService — the same reasoning
// internal/catalog/httpapi's catalogService interface documents: it lets
// handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// catalogService is: Go interfaces are satisfied structurally, so nothing
// outside this package needs to name it.
type productService interface {
	Get(ctx context.Context, id uuid.UUID) (product.Product, error)
	List(ctx context.Context) ([]product.Product, error)
	Create(ctx context.Context, p product.Product) (product.Product, error)
	Update(ctx context.Context, p product.Product) (product.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProductHandler serves the Product REST endpoints:
//
//	POST   /api/v1/products
//	GET    /api/v1/products
//	GET    /api/v1/products/{id}
//	PUT    /api/v1/products/{id}
//	DELETE /api/v1/products/{id}
//
// It depends only on productService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is ProductService's job.
type ProductHandler struct {
	products productService
}

// NewProductHandler builds a ProductHandler.
func NewProductHandler(products productService) *ProductHandler {
	return &ProductHandler{products: products}
}

// Create handles POST /api/v1/products.
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.products.Create(r.Context(), req.toProduct(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newProductResponse(created))
}

// List handles GET /api/v1/products.
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.products.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProductListResponse(products))
}

// Get handles GET /api/v1/products/{id}.
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	p, err := h.products.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProductResponse(p))
}

// Update handles PUT /api/v1/products/{id}.
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req productRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.products.Update(r.Context(), req.toProduct(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newProductResponse(updated))
}

// Delete handles DELETE /api/v1/products/{id}.
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.products.Delete(r.Context(), id); err != nil {
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
