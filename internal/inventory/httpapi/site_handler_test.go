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

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeSiteService is the seam httpapi.SiteHandler depends on (see its
// unexported siteService interface in site_handler.go). It lets these
// tests exercise HTTP-only concerns — status codes, JSON shapes, routing,
// error translation — without a real service, repository, or database;
// internal/inventory/service and internal/inventory/postgres each have
// their own tests for the layers below this one.
type fakeSiteService struct {
	sites map[uuid.UUID]inventory.Site
	err   error // if set, every method returns this error instead
}

func newFakeSiteService(sites ...inventory.Site) *fakeSiteService {
	f := &fakeSiteService{sites: make(map[uuid.UUID]inventory.Site)}
	for _, s := range sites {
		f.sites[s.ID] = s
	}
	return f
}

func (f *fakeSiteService) Get(_ context.Context, id uuid.UUID) (inventory.Site, error) {
	if f.err != nil {
		return inventory.Site{}, f.err
	}
	s, ok := f.sites[id]
	if !ok {
		return inventory.Site{}, apperror.NotFound("site not found")
	}
	return s, nil
}

func (f *fakeSiteService) List(context.Context) ([]inventory.Site, error) {
	if f.err != nil {
		return nil, f.err
	}
	sites := make([]inventory.Site, 0, len(f.sites))
	for _, s := range f.sites {
		sites = append(sites, s)
	}
	return sites, nil
}

func (f *fakeSiteService) Create(_ context.Context, site inventory.Site) (inventory.Site, error) {
	if f.err != nil {
		return inventory.Site{}, f.err
	}
	site.ID = uuid.New()
	site.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	site.UpdatedAt = site.CreatedAt
	f.sites[site.ID] = site
	return site, nil
}

func (f *fakeSiteService) Update(_ context.Context, site inventory.Site) (inventory.Site, error) {
	if f.err != nil {
		return inventory.Site{}, f.err
	}
	if _, ok := f.sites[site.ID]; !ok {
		return inventory.Site{}, apperror.NotFound("site not found")
	}
	f.sites[site.ID] = site
	return site, nil
}

func (f *fakeSiteService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.sites[id]; !ok {
		return apperror.NotFound("site not found")
	}
	delete(f.sites, id)
	return nil
}

// newTestRouter mounts a SiteHandler backed by svc on a real chi.Router,
// so tests that need a URL path parameter (Get/Update/Delete's {id}) get
// one populated the same way production code does, rather than faking
// chi's route context by hand.
func newTestRouter(svc *fakeSiteService) http.Handler {
	handler := httpapi.NewSiteHandler(svc)

	r := chi.NewRouter()
	r.Post("/sites", handler.Create)
	r.Get("/sites", handler.List)
	r.Get("/sites/{id}", handler.Get)
	r.Put("/sites/{id}", handler.Update)
	r.Delete("/sites/{id}", handler.Delete)
	return r
}

func TestSiteHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeSiteService())

	req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Main Office","description":"HQ"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "Main Office" || body.Description != "HQ" {
		t.Errorf("body = %+v, want Name=Main Office Description=HQ", body)
	}
}

func TestSiteHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeSiteService())

	req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSiteHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeSiteService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestSiteHandlerList(t *testing.T) {
	a := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}}
	b := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}}
	router := newTestRouter(newFakeSiteService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/sites", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Sites []struct {
			ID string `json:"id"`
		} `json:"sites"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Sites) != 2 {
		t.Fatalf("len(sites) = %d, want 2", len(body.Sites))
	}
}

func TestSiteHandlerGet(t *testing.T) {
	site := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Main Office"}}
	router := newTestRouter(newFakeSiteService(site))

	req := httptest.NewRequest(http.MethodGet, "/sites/"+site.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestSiteHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeSiteService())

	req := httptest.NewRequest(http.MethodGet, "/sites/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSiteHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeSiteService())

	req := httptest.NewRequest(http.MethodGet, "/sites/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSiteHandlerUpdate(t *testing.T) {
	site := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}}
	router := newTestRouter(newFakeSiteService(site))

	req := httptest.NewRequest(http.MethodPut, "/sites/"+site.ID.String(), strings.NewReader(`{"name":"New Name"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" {
		t.Errorf("Name = %q, want %q", body.Name, "New Name")
	}
}

func TestSiteHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeSiteService())

	req := httptest.NewRequest(http.MethodPut, "/sites/"+uuid.New().String(), strings.NewReader(`{"name":"New Name"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSiteHandlerDelete(t *testing.T) {
	site := inventory.Site{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}}
	router := newTestRouter(newFakeSiteService(site))

	req := httptest.NewRequest(http.MethodDelete, "/sites/"+site.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestSiteHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeSiteService())

	req := httptest.NewRequest(http.MethodDelete, "/sites/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
