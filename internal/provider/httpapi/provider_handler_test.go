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
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
	"github.com/paladindigitalgh/palladium-oss/internal/provider/httpapi"
)

// fakeProviderService is the seam httpapi.ProviderHandler depends on
// (see its unexported providerService interface in
// provider_handler.go). It lets these tests exercise HTTP-only
// concerns — status codes, JSON shapes, routing, error translation —
// without a real service, repository, or database;
// internal/provider/service and internal/provider/postgres each have
// their own tests for the layers below this one.
type fakeProviderService struct {
	providers map[uuid.UUID]provider.Provider
	err       error // if set, every method returns this error instead
}

func newFakeProviderService(providers ...provider.Provider) *fakeProviderService {
	f := &fakeProviderService{providers: make(map[uuid.UUID]provider.Provider)}
	for _, p := range providers {
		f.providers[p.ID] = p
	}
	return f
}

func (f *fakeProviderService) Get(_ context.Context, id uuid.UUID) (provider.Provider, error) {
	if f.err != nil {
		return provider.Provider{}, f.err
	}
	p, ok := f.providers[id]
	if !ok {
		return provider.Provider{}, apperror.NotFound("provider not found")
	}
	return p, nil
}

func (f *fakeProviderService) List(context.Context) ([]provider.Provider, error) {
	if f.err != nil {
		return nil, f.err
	}
	providers := make([]provider.Provider, 0, len(f.providers))
	for _, p := range f.providers {
		providers = append(providers, p)
	}
	return providers, nil
}

func (f *fakeProviderService) Create(_ context.Context, p provider.Provider) (provider.Provider, error) {
	if f.err != nil {
		return provider.Provider{}, f.err
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.UpdatedAt = p.CreatedAt
	f.providers[p.ID] = p
	return p, nil
}

func (f *fakeProviderService) Update(_ context.Context, p provider.Provider) (provider.Provider, error) {
	if f.err != nil {
		return provider.Provider{}, f.err
	}
	if _, ok := f.providers[p.ID]; !ok {
		return provider.Provider{}, apperror.NotFound("provider not found")
	}
	f.providers[p.ID] = p
	return p, nil
}

func (f *fakeProviderService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.providers[id]; !ok {
		return apperror.NotFound("provider not found")
	}
	delete(f.providers, id)
	return nil
}

// newTestRouter mounts a ProviderHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeProviderService) http.Handler {
	handler := httpapi.NewProviderHandler(svc)

	r := chi.NewRouter()
	r.Post("/providers", handler.Create)
	r.Get("/providers", handler.List)
	r.Get("/providers/{id}", handler.Get)
	r.Put("/providers/{id}", handler.Update)
	r.Delete("/providers/{id}", handler.Delete)
	return r
}

const validBody = `{"name":"Acme Fiber","status":"Active"}`

func TestProviderHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeProviderService())

	req := httptest.NewRequest(http.MethodPost, "/providers", strings.NewReader(validBody))
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
	if body.Name != "Acme Fiber" {
		t.Errorf("name = %q, want %q", body.Name, "Acme Fiber")
	}
	if body.Status != "Active" {
		t.Errorf("status = %q, want %q", body.Status, "Active")
	}
}

func TestProviderHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeProviderService())

	req := httptest.NewRequest(http.MethodPost, "/providers", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProviderHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeProviderService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/providers", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestProviderHandlerList(t *testing.T) {
	a := provider.Provider{ID: uuid.New(), Name: "Alpha", Status: provider.StatusActive}
	b := provider.Provider{ID: uuid.New(), Name: "Beta", Status: provider.StatusActive}
	router := newTestRouter(newFakeProviderService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/providers", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Providers []struct {
			ID string `json:"id"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Providers) != 2 {
		t.Fatalf("len(providers) = %d, want 2", len(body.Providers))
	}
}

func TestProviderHandlerGet(t *testing.T) {
	p := provider.Provider{ID: uuid.New(), Name: "Alpha", Status: provider.StatusActive}
	router := newTestRouter(newFakeProviderService(p))

	req := httptest.NewRequest(http.MethodGet, "/providers/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProviderHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeProviderService())

	req := httptest.NewRequest(http.MethodGet, "/providers/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProviderHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeProviderService())

	req := httptest.NewRequest(http.MethodGet, "/providers/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProviderHandlerUpdate(t *testing.T) {
	p := provider.Provider{ID: uuid.New(), Name: "Alpha", Status: provider.StatusActive}
	router := newTestRouter(newFakeProviderService(p))

	req := httptest.NewRequest(http.MethodPut, "/providers/"+p.ID.String(),
		strings.NewReader(`{"name":"Beta","status":"Inactive"}`))
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
	if body.Name != "Beta" {
		t.Errorf("name = %q, want %q", body.Name, "Beta")
	}
	if body.Status != "Inactive" {
		t.Errorf("status = %q, want %q", body.Status, "Inactive")
	}
}

func TestProviderHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeProviderService())

	req := httptest.NewRequest(http.MethodPut, "/providers/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProviderHandlerDelete(t *testing.T) {
	p := provider.Provider{ID: uuid.New(), Name: "Alpha", Status: provider.StatusActive}
	router := newTestRouter(newFakeProviderService(p))

	req := httptest.NewRequest(http.MethodDelete, "/providers/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestProviderHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeProviderService())

	req := httptest.NewRequest(http.MethodDelete, "/providers/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
