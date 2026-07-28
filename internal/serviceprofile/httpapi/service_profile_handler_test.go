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
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/httpapi"
)

// fakeServiceProfileService is the seam httpapi.ServiceProfileHandler
// depends on (see its unexported serviceProfileService interface in
// service_profile_handler.go). It lets these tests exercise HTTP-only
// concerns — status codes, JSON shapes, routing, error translation —
// without a real service, repository, or database;
// internal/serviceprofile/service and internal/serviceprofile/postgres
// each have their own tests for the layers below this one.
type fakeServiceProfileService struct {
	profiles map[uuid.UUID]serviceprofile.ServiceProfile
	err      error // if set, every method returns this error instead
}

func newFakeServiceProfileService(profiles ...serviceprofile.ServiceProfile) *fakeServiceProfileService {
	f := &fakeServiceProfileService{profiles: make(map[uuid.UUID]serviceprofile.ServiceProfile)}
	for _, p := range profiles {
		f.profiles[p.ID] = p
	}
	return f
}

func (f *fakeServiceProfileService) Get(_ context.Context, id uuid.UUID) (serviceprofile.ServiceProfile, error) {
	if f.err != nil {
		return serviceprofile.ServiceProfile{}, f.err
	}
	p, ok := f.profiles[id]
	if !ok {
		return serviceprofile.ServiceProfile{}, apperror.NotFound("service profile not found")
	}
	return p, nil
}

func (f *fakeServiceProfileService) List(context.Context) ([]serviceprofile.ServiceProfile, error) {
	if f.err != nil {
		return nil, f.err
	}
	profiles := make([]serviceprofile.ServiceProfile, 0, len(f.profiles))
	for _, p := range f.profiles {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (f *fakeServiceProfileService) Create(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	if f.err != nil {
		return serviceprofile.ServiceProfile{}, f.err
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.UpdatedAt = p.CreatedAt
	f.profiles[p.ID] = p
	return p, nil
}

func (f *fakeServiceProfileService) Update(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	if f.err != nil {
		return serviceprofile.ServiceProfile{}, f.err
	}
	if _, ok := f.profiles[p.ID]; !ok {
		return serviceprofile.ServiceProfile{}, apperror.NotFound("service profile not found")
	}
	f.profiles[p.ID] = p
	return p, nil
}

func (f *fakeServiceProfileService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.profiles[id]; !ok {
		return apperror.NotFound("service profile not found")
	}
	delete(f.profiles, id)
	return nil
}

// newTestRouter mounts a ServiceProfileHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeServiceProfileService) http.Handler {
	handler := httpapi.NewServiceProfileHandler(svc)

	r := chi.NewRouter()
	r.Post("/service-profiles", handler.Create)
	r.Get("/service-profiles", handler.List)
	r.Get("/service-profiles/{id}", handler.Get)
	r.Put("/service-profiles/{id}", handler.Update)
	r.Delete("/service-profiles/{id}", handler.Delete)
	return r
}

const validBody = `{"name":"Residential Internet","status":"Active"}`

func TestServiceProfileHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeServiceProfileService())

	req := httptest.NewRequest(http.MethodPost, "/service-profiles", strings.NewReader(validBody))
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
	if body.Name != "Residential Internet" {
		t.Errorf("name = %q, want %q", body.Name, "Residential Internet")
	}
	if body.Status != "Active" {
		t.Errorf("status = %q, want %q", body.Status, "Active")
	}
}

func TestServiceProfileHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeServiceProfileService())

	req := httptest.NewRequest(http.MethodPost, "/service-profiles", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceProfileHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeServiceProfileService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/service-profiles", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestServiceProfileHandlerList(t *testing.T) {
	a := serviceprofile.ServiceProfile{ID: uuid.New(), Name: "Alpha", Status: serviceprofile.StatusActive}
	b := serviceprofile.ServiceProfile{ID: uuid.New(), Name: "Beta", Status: serviceprofile.StatusActive}
	router := newTestRouter(newFakeServiceProfileService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/service-profiles", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		ServiceProfiles []struct {
			ID string `json:"id"`
		} `json:"service_profiles"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.ServiceProfiles) != 2 {
		t.Fatalf("len(service_profiles) = %d, want 2", len(body.ServiceProfiles))
	}
}

func TestServiceProfileHandlerGet(t *testing.T) {
	p := serviceprofile.ServiceProfile{ID: uuid.New(), Name: "Alpha", Status: serviceprofile.StatusActive}
	router := newTestRouter(newFakeServiceProfileService(p))

	req := httptest.NewRequest(http.MethodGet, "/service-profiles/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestServiceProfileHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceProfileService())

	req := httptest.NewRequest(http.MethodGet, "/service-profiles/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServiceProfileHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeServiceProfileService())

	req := httptest.NewRequest(http.MethodGet, "/service-profiles/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceProfileHandlerUpdate(t *testing.T) {
	p := serviceprofile.ServiceProfile{ID: uuid.New(), Name: "Alpha", Status: serviceprofile.StatusActive}
	router := newTestRouter(newFakeServiceProfileService(p))

	req := httptest.NewRequest(http.MethodPut, "/service-profiles/"+p.ID.String(),
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

func TestServiceProfileHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceProfileService())

	req := httptest.NewRequest(http.MethodPut, "/service-profiles/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServiceProfileHandlerDelete(t *testing.T) {
	p := serviceprofile.ServiceProfile{ID: uuid.New(), Name: "Alpha", Status: serviceprofile.StatusActive}
	router := newTestRouter(newFakeServiceProfileService(p))

	req := httptest.NewRequest(http.MethodDelete, "/service-profiles/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestServiceProfileHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceProfileService())

	req := httptest.NewRequest(http.MethodDelete, "/service-profiles/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
