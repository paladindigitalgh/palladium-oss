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

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAccessNetworkService is the seam httpapi.AccessNetworkHandler
// depends on (see its unexported accessNetworkService interface in
// access_network_handler.go). It lets these tests exercise HTTP-only
// concerns — status codes, JSON shapes, routing, error translation —
// without a real service, repository, or database;
// internal/accessnetwork/service and internal/accessnetwork/postgres
// each have their own tests for the layers below this one.
type fakeAccessNetworkService struct {
	networks map[uuid.UUID]accessnetwork.AccessNetwork
	err      error // if set, every method returns this error instead
}

func newFakeAccessNetworkService(networks ...accessnetwork.AccessNetwork) *fakeAccessNetworkService {
	f := &fakeAccessNetworkService{networks: make(map[uuid.UUID]accessnetwork.AccessNetwork)}
	for _, a := range networks {
		f.networks[a.ID] = a
	}
	return f
}

func (f *fakeAccessNetworkService) Get(_ context.Context, id uuid.UUID) (accessnetwork.AccessNetwork, error) {
	if f.err != nil {
		return accessnetwork.AccessNetwork{}, f.err
	}
	a, ok := f.networks[id]
	if !ok {
		return accessnetwork.AccessNetwork{}, apperror.NotFound("access network not found")
	}
	return a, nil
}

func (f *fakeAccessNetworkService) List(context.Context) ([]accessnetwork.AccessNetwork, error) {
	if f.err != nil {
		return nil, f.err
	}
	networks := make([]accessnetwork.AccessNetwork, 0, len(f.networks))
	for _, a := range f.networks {
		networks = append(networks, a)
	}
	return networks, nil
}

func (f *fakeAccessNetworkService) Create(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	if f.err != nil {
		return accessnetwork.AccessNetwork{}, f.err
	}
	a.ID = uuid.New()
	a.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.UpdatedAt = a.CreatedAt
	f.networks[a.ID] = a
	return a, nil
}

func (f *fakeAccessNetworkService) Update(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	if f.err != nil {
		return accessnetwork.AccessNetwork{}, f.err
	}
	if _, ok := f.networks[a.ID]; !ok {
		return accessnetwork.AccessNetwork{}, apperror.NotFound("access network not found")
	}
	f.networks[a.ID] = a
	return a, nil
}

func (f *fakeAccessNetworkService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.networks[id]; !ok {
		return apperror.NotFound("access network not found")
	}
	delete(f.networks, id)
	return nil
}

// newTestRouter mounts an AccessNetworkHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeAccessNetworkService) http.Handler {
	handler := httpapi.NewAccessNetworkHandler(svc)

	r := chi.NewRouter()
	r.Post("/access-networks", handler.Create)
	r.Get("/access-networks", handler.List)
	r.Get("/access-networks/{id}", handler.Get)
	r.Put("/access-networks/{id}", handler.Update)
	r.Delete("/access-networks/{id}", handler.Delete)
	return r
}

const validBody = `{"name":"North Region GPON","status":"Active"}`

func TestAccessNetworkHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeAccessNetworkService())

	req := httptest.NewRequest(http.MethodPost, "/access-networks", strings.NewReader(validBody))
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
	if body.Name != "North Region GPON" || body.Status != "Active" {
		t.Errorf("body = %+v, want Name=North Region GPON Status=Active", body)
	}
}

func TestAccessNetworkHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeAccessNetworkService())

	req := httptest.NewRequest(http.MethodPost, "/access-networks", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAccessNetworkHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeAccessNetworkService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/access-networks", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAccessNetworkHandlerList(t *testing.T) {
	a := accessnetwork.AccessNetwork{ID: uuid.New(), Name: "A", Status: accessnetwork.AccessNetworkStatusActive}
	b := accessnetwork.AccessNetwork{ID: uuid.New(), Name: "B", Status: accessnetwork.AccessNetworkStatusActive}
	router := newTestRouter(newFakeAccessNetworkService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/access-networks", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		AccessNetworks []struct {
			ID string `json:"id"`
		} `json:"access_networks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.AccessNetworks) != 2 {
		t.Fatalf("len(access_networks) = %d, want 2", len(body.AccessNetworks))
	}
}

func TestAccessNetworkHandlerGet(t *testing.T) {
	a := accessnetwork.AccessNetwork{ID: uuid.New(), Name: "North Region GPON", Status: accessnetwork.AccessNetworkStatusActive}
	router := newTestRouter(newFakeAccessNetworkService(a))

	req := httptest.NewRequest(http.MethodGet, "/access-networks/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestAccessNetworkHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessNetworkService())

	req := httptest.NewRequest(http.MethodGet, "/access-networks/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAccessNetworkHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeAccessNetworkService())

	req := httptest.NewRequest(http.MethodGet, "/access-networks/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAccessNetworkHandlerUpdate(t *testing.T) {
	a := accessnetwork.AccessNetwork{ID: uuid.New(), Name: "Old Name", Status: accessnetwork.AccessNetworkStatusActive}
	router := newTestRouter(newFakeAccessNetworkService(a))

	req := httptest.NewRequest(http.MethodPut, "/access-networks/"+a.ID.String(),
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

func TestAccessNetworkHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessNetworkService())

	req := httptest.NewRequest(http.MethodPut, "/access-networks/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAccessNetworkHandlerDelete(t *testing.T) {
	a := accessnetwork.AccessNetwork{ID: uuid.New(), Name: "Temporary", Status: accessnetwork.AccessNetworkStatusActive}
	router := newTestRouter(newFakeAccessNetworkService(a))

	req := httptest.NewRequest(http.MethodDelete, "/access-networks/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestAccessNetworkHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessNetworkService())

	req := httptest.NewRequest(http.MethodDelete, "/access-networks/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
