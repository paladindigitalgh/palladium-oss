package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	"github.com/paladindigitalgh/palladium-oss/internal/product/httpapi"
)

// fakeProductService is the seam httpapi.ProductHandler depends on (see
// its unexported productService interface in product_handler.go). It lets
// these tests exercise HTTP-only concerns — status codes, JSON shapes,
// routing, error translation — without a real service, repository, or
// database; internal/product/service and internal/product/postgres each
// have their own tests for the layers below this one.
type fakeProductService struct {
	products map[uuid.UUID]product.Product
	err      error // if set, every method returns this error instead
}

func newFakeProductService(products ...product.Product) *fakeProductService {
	f := &fakeProductService{products: make(map[uuid.UUID]product.Product)}
	for _, p := range products {
		f.products[p.ID] = p
	}
	return f
}

func (f *fakeProductService) Get(_ context.Context, id uuid.UUID) (product.Product, error) {
	if f.err != nil {
		return product.Product{}, f.err
	}
	p, ok := f.products[id]
	if !ok {
		return product.Product{}, apperror.NotFound("product not found")
	}
	return p, nil
}

func (f *fakeProductService) List(context.Context) ([]product.Product, error) {
	if f.err != nil {
		return nil, f.err
	}
	products := make([]product.Product, 0, len(f.products))
	for _, p := range f.products {
		products = append(products, p)
	}
	return products, nil
}

func (f *fakeProductService) Create(_ context.Context, p product.Product) (product.Product, error) {
	if f.err != nil {
		return product.Product{}, f.err
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.UpdatedAt = p.CreatedAt
	f.products[p.ID] = p
	return p, nil
}

func (f *fakeProductService) Update(_ context.Context, p product.Product) (product.Product, error) {
	if f.err != nil {
		return product.Product{}, f.err
	}
	if _, ok := f.products[p.ID]; !ok {
		return product.Product{}, apperror.NotFound("product not found")
	}
	f.products[p.ID] = p
	return p, nil
}

func (f *fakeProductService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.products[id]; !ok {
		return apperror.NotFound("product not found")
	}
	delete(f.products, id)
	return nil
}

// newTestRouter mounts a ProductHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeProductService) http.Handler {
	handler := httpapi.NewProductHandler(svc)

	r := chi.NewRouter()
	r.Post("/products", handler.Create)
	r.Get("/products", handler.List)
	r.Get("/products/{id}", handler.Get)
	r.Put("/products/{id}", handler.Update)
	r.Delete("/products/{id}", handler.Delete)
	return r
}

const validBody = `{"catalog_id":"11111111-1111-1111-1111-111111111111","name":"Residential Internet 100/20","category":"Internet","status":"Active"}`

func TestProductHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeProductService())

	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID        string `json:"id"`
		CatalogID string `json:"catalog_id"`
		Name      string `json:"name"`
		Category  string `json:"category"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.CatalogID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("catalog_id = %q, want %q", body.CatalogID, "11111111-1111-1111-1111-111111111111")
	}
	if body.Name != "Residential Internet 100/20" || body.Category != "Internet" || body.Status != "Active" {
		t.Errorf("body = %+v, want Name=Residential Internet 100/20 Category=Internet Status=Active", body)
	}
}

func TestProductHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeProductService())

	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProductHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeProductService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestProductHandlerCreatePropagatesConflictOnUnknownCatalog(t *testing.T) {
	svc := newFakeProductService()
	svc.err = apperror.Conflict("create product: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestProductHandlerList(t *testing.T) {
	a := product.Product{ID: uuid.New(), CatalogID: uuid.New(), Name: "A", Category: product.ProductCategoryInternet, Status: product.ProductStatusActive}
	b := product.Product{ID: uuid.New(), CatalogID: uuid.New(), Name: "B", Category: product.ProductCategoryVoice, Status: product.ProductStatusActive}
	router := newTestRouter(newFakeProductService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Products []struct {
			ID string `json:"id"`
		} `json:"products"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(body.Products))
	}
}

func TestProductHandlerGet(t *testing.T) {
	p := product.Product{ID: uuid.New(), CatalogID: uuid.New(), Name: "Residential Internet", Category: product.ProductCategoryInternet, Status: product.ProductStatusActive}
	router := newTestRouter(newFakeProductService(p))

	req := httptest.NewRequest(http.MethodGet, "/products/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProductHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeProductService())

	req := httptest.NewRequest(http.MethodGet, "/products/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProductHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeProductService())

	req := httptest.NewRequest(http.MethodGet, "/products/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProductHandlerUpdate(t *testing.T) {
	p := product.Product{ID: uuid.New(), CatalogID: uuid.New(), Name: "Old Name", Category: product.ProductCategoryInternet, Status: product.ProductStatusActive}
	router := newTestRouter(newFakeProductService(p))

	req := httptest.NewRequest(http.MethodPut, "/products/"+p.ID.String(),
		strings.NewReader(`{"catalog_id":"`+p.CatalogID.String()+`","name":"New Name","category":"Voice","status":"Retired"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name     string `json:"name"`
		Category string `json:"category"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" || body.Category != "Voice" || body.Status != "Retired" {
		t.Errorf("body = %+v, want Name=New Name Category=Voice Status=Retired", body)
	}
}

func TestProductHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeProductService())

	req := httptest.NewRequest(http.MethodPut, "/products/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProductHandlerDelete(t *testing.T) {
	p := product.Product{ID: uuid.New(), CatalogID: uuid.New(), Name: "Temporary", Category: product.ProductCategoryInternet, Status: product.ProductStatusActive}
	router := newTestRouter(newFakeProductService(p))

	req := httptest.NewRequest(http.MethodDelete, "/products/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestProductHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeProductService())

	req := httptest.NewRequest(http.MethodDelete, "/products/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
