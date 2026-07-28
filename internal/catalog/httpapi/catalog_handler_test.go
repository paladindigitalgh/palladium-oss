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

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	"github.com/paladindigitalgh/palladium-oss/internal/catalog/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeCatalogService is the seam httpapi.CatalogHandler depends on (see
// its unexported catalogService interface in catalog_handler.go). It lets
// these tests exercise HTTP-only concerns — status codes, JSON shapes,
// routing, error translation — without a real service, repository, or
// database; internal/catalog/service and internal/catalog/postgres each
// have their own tests for the layers below this one.
type fakeCatalogService struct {
	catalogs map[uuid.UUID]catalog.ProductCatalog
	err      error // if set, every method returns this error instead
}

func newFakeCatalogService(catalogs ...catalog.ProductCatalog) *fakeCatalogService {
	f := &fakeCatalogService{catalogs: make(map[uuid.UUID]catalog.ProductCatalog)}
	for _, c := range catalogs {
		f.catalogs[c.ID] = c
	}
	return f
}

func (f *fakeCatalogService) Get(_ context.Context, id uuid.UUID) (catalog.ProductCatalog, error) {
	if f.err != nil {
		return catalog.ProductCatalog{}, f.err
	}
	c, ok := f.catalogs[id]
	if !ok {
		return catalog.ProductCatalog{}, apperror.NotFound("catalog not found")
	}
	return c, nil
}

func (f *fakeCatalogService) List(context.Context) ([]catalog.ProductCatalog, error) {
	if f.err != nil {
		return nil, f.err
	}
	catalogs := make([]catalog.ProductCatalog, 0, len(f.catalogs))
	for _, c := range f.catalogs {
		catalogs = append(catalogs, c)
	}
	return catalogs, nil
}

func (f *fakeCatalogService) Create(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	if f.err != nil {
		return catalog.ProductCatalog{}, f.err
	}
	c.ID = uuid.New()
	c.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.UpdatedAt = c.CreatedAt
	f.catalogs[c.ID] = c
	return c, nil
}

func (f *fakeCatalogService) Update(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	if f.err != nil {
		return catalog.ProductCatalog{}, f.err
	}
	if _, ok := f.catalogs[c.ID]; !ok {
		return catalog.ProductCatalog{}, apperror.NotFound("catalog not found")
	}
	f.catalogs[c.ID] = c
	return c, nil
}

func (f *fakeCatalogService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.catalogs[id]; !ok {
		return apperror.NotFound("catalog not found")
	}
	delete(f.catalogs, id)
	return nil
}

// newTestRouter mounts a CatalogHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeCatalogService) http.Handler {
	handler := httpapi.NewCatalogHandler(svc)

	r := chi.NewRouter()
	r.Post("/catalogs", handler.Create)
	r.Get("/catalogs", handler.List)
	r.Get("/catalogs/{id}", handler.Get)
	r.Put("/catalogs/{id}", handler.Update)
	r.Delete("/catalogs/{id}", handler.Delete)
	return r
}

const validBody = `{"name":"Residential","status":"Active"}`

func TestCatalogHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeCatalogService())

	req := httptest.NewRequest(http.MethodPost, "/catalogs", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "Residential" || body.Status != "Active" {
		t.Errorf("body = %+v, want Name=Residential Status=Active", body)
	}
}

func TestCatalogHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeCatalogService())

	req := httptest.NewRequest(http.MethodPost, "/catalogs", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCatalogHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeCatalogService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/catalogs", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCatalogHandlerList(t *testing.T) {
	a := catalog.ProductCatalog{ID: uuid.New(), Name: "A", Status: catalog.CatalogStatusActive}
	b := catalog.ProductCatalog{ID: uuid.New(), Name: "B", Status: catalog.CatalogStatusActive}
	router := newTestRouter(newFakeCatalogService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/catalogs", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Catalogs []struct {
			ID string `json:"id"`
		} `json:"catalogs"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Catalogs) != 2 {
		t.Fatalf("len(catalogs) = %d, want 2", len(body.Catalogs))
	}
}

func TestCatalogHandlerGet(t *testing.T) {
	c := catalog.ProductCatalog{ID: uuid.New(), Name: "Residential", Status: catalog.CatalogStatusActive}
	router := newTestRouter(newFakeCatalogService(c))

	req := httptest.NewRequest(http.MethodGet, "/catalogs/"+c.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestCatalogHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeCatalogService())

	req := httptest.NewRequest(http.MethodGet, "/catalogs/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCatalogHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeCatalogService())

	req := httptest.NewRequest(http.MethodGet, "/catalogs/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCatalogHandlerUpdate(t *testing.T) {
	c := catalog.ProductCatalog{ID: uuid.New(), Name: "Old Name", Status: catalog.CatalogStatusActive}
	router := newTestRouter(newFakeCatalogService(c))

	req := httptest.NewRequest(http.MethodPut, "/catalogs/"+c.ID.String(),
		strings.NewReader(`{"name":"New Name","status":"Inactive"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" || body.Status != "Inactive" {
		t.Errorf("body = %+v, want Name=New Name Status=Inactive", body)
	}
}

func TestCatalogHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeCatalogService())

	req := httptest.NewRequest(http.MethodPut, "/catalogs/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCatalogHandlerDelete(t *testing.T) {
	c := catalog.ProductCatalog{ID: uuid.New(), Name: "Temporary", Status: catalog.CatalogStatusActive}
	router := newTestRouter(newFakeCatalogService(c))

	req := httptest.NewRequest(http.MethodDelete, "/catalogs/"+c.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestCatalogHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeCatalogService())

	req := httptest.NewRequest(http.MethodDelete, "/catalogs/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
