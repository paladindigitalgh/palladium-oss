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
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/service/httpapi"
)

// fakeServiceService is the seam httpapi.ServiceHandler depends on (see
// its unexported serviceService interface in service_handler.go). It lets
// these tests exercise HTTP-only concerns — status codes, JSON shapes,
// routing, error translation — without a real service, repository, or
// database; internal/service/service and internal/service/postgres each
// have their own tests for the layers below this one.
type fakeServiceService struct {
	services map[uuid.UUID]domainservice.Service
	err      error // if set, every method returns this error instead
}

func newFakeServiceService(services ...domainservice.Service) *fakeServiceService {
	f := &fakeServiceService{services: make(map[uuid.UUID]domainservice.Service)}
	for _, s := range services {
		f.services[s.ID] = s
	}
	return f
}

func (f *fakeServiceService) Get(_ context.Context, id uuid.UUID) (domainservice.Service, error) {
	if f.err != nil {
		return domainservice.Service{}, f.err
	}
	s, ok := f.services[id]
	if !ok {
		return domainservice.Service{}, apperror.NotFound("service not found")
	}
	return s, nil
}

func (f *fakeServiceService) List(context.Context) ([]domainservice.Service, error) {
	if f.err != nil {
		return nil, f.err
	}
	services := make([]domainservice.Service, 0, len(f.services))
	for _, s := range f.services {
		services = append(services, s)
	}
	return services, nil
}

func (f *fakeServiceService) Create(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	if f.err != nil {
		return domainservice.Service{}, f.err
	}
	s.ID = uuid.New()
	s.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.UpdatedAt = s.CreatedAt
	f.services[s.ID] = s
	return s, nil
}

func (f *fakeServiceService) Update(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	if f.err != nil {
		return domainservice.Service{}, f.err
	}
	if _, ok := f.services[s.ID]; !ok {
		return domainservice.Service{}, apperror.NotFound("service not found")
	}
	f.services[s.ID] = s
	return s, nil
}

func (f *fakeServiceService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.services[id]; !ok {
		return apperror.NotFound("service not found")
	}
	delete(f.services, id)
	return nil
}

// newTestRouter mounts a ServiceHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeServiceService) http.Handler {
	handler := httpapi.NewServiceHandler(svc)

	r := chi.NewRouter()
	r.Post("/services", handler.Create)
	r.Get("/services", handler.List)
	r.Get("/services/{id}", handler.Get)
	r.Put("/services/{id}", handler.Update)
	r.Delete("/services/{id}", handler.Delete)
	return r
}

const validBody = `{"location_id":"11111111-1111-1111-1111-111111111111","product_id":"22222222-2222-2222-2222-222222222222","status":"Pending"}`

func TestServiceHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeServiceService())

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		LocationID string `json:"location_id"`
		ProductID  string `json:"product_id"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.LocationID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("location_id = %q, want %q", body.LocationID, "11111111-1111-1111-1111-111111111111")
	}
	if body.ProductID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("product_id = %q, want %q", body.ProductID, "22222222-2222-2222-2222-222222222222")
	}
	if body.Status != "Pending" {
		t.Errorf("status = %q, want %q", body.Status, "Pending")
	}
}

func TestServiceHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeServiceService())

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeServiceService()
	svc.err = apperror.Invalid("status: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(`{"status":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestServiceHandlerCreatePropagatesConflictOnUnknownLocationOrProduct(t *testing.T) {
	svc := newFakeServiceService()
	svc.err = apperror.Conflict("create service: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestServiceHandlerList(t *testing.T) {
	a := domainservice.Service{ID: uuid.New(), LocationID: uuid.New(), ProductID: uuid.New(), Status: domainservice.ServiceStatusPending}
	b := domainservice.Service{ID: uuid.New(), LocationID: uuid.New(), ProductID: uuid.New(), Status: domainservice.ServiceStatusActive}
	router := newTestRouter(newFakeServiceService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Services []struct {
			ID string `json:"id"`
		} `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Services) != 2 {
		t.Fatalf("len(services) = %d, want 2", len(body.Services))
	}
}

func TestServiceHandlerGet(t *testing.T) {
	s := domainservice.Service{ID: uuid.New(), LocationID: uuid.New(), ProductID: uuid.New(), Status: domainservice.ServiceStatusPending}
	router := newTestRouter(newFakeServiceService(s))

	req := httptest.NewRequest(http.MethodGet, "/services/"+s.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestServiceHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceService())

	req := httptest.NewRequest(http.MethodGet, "/services/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServiceHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeServiceService())

	req := httptest.NewRequest(http.MethodGet, "/services/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceHandlerUpdate(t *testing.T) {
	s := domainservice.Service{ID: uuid.New(), LocationID: uuid.New(), ProductID: uuid.New(), Status: domainservice.ServiceStatusPending}
	router := newTestRouter(newFakeServiceService(s))

	req := httptest.NewRequest(http.MethodPut, "/services/"+s.ID.String(),
		strings.NewReader(`{"location_id":"`+s.LocationID.String()+`","product_id":"`+s.ProductID.String()+`","status":"Active"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "Active" {
		t.Errorf("status = %q, want %q", body.Status, "Active")
	}
}

func TestServiceHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceService())

	req := httptest.NewRequest(http.MethodPut, "/services/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServiceHandlerDelete(t *testing.T) {
	s := domainservice.Service{ID: uuid.New(), LocationID: uuid.New(), ProductID: uuid.New(), Status: domainservice.ServiceStatusPending}
	router := newTestRouter(newFakeServiceService(s))

	req := httptest.NewRequest(http.MethodDelete, "/services/"+s.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestServiceHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceService())

	req := httptest.NewRequest(http.MethodDelete, "/services/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
