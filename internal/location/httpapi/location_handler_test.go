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

	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeLocationService is the seam httpapi.LocationHandler depends on (see
// its unexported locationService interface in location_handler.go). It
// lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, routing, error translation — without a real service,
// repository, or database; internal/location/service and
// internal/location/postgres each have their own tests for the layers
// below this one.
type fakeLocationService struct {
	locations map[uuid.UUID]location.Location
	err       error // if set, every method returns this error instead
}

func newFakeLocationService(locations ...location.Location) *fakeLocationService {
	f := &fakeLocationService{locations: make(map[uuid.UUID]location.Location)}
	for _, l := range locations {
		f.locations[l.ID] = l
	}
	return f
}

func (f *fakeLocationService) Get(_ context.Context, id uuid.UUID) (location.Location, error) {
	if f.err != nil {
		return location.Location{}, f.err
	}
	l, ok := f.locations[id]
	if !ok {
		return location.Location{}, apperror.NotFound("location not found")
	}
	return l, nil
}

func (f *fakeLocationService) List(context.Context) ([]location.Location, error) {
	if f.err != nil {
		return nil, f.err
	}
	locations := make([]location.Location, 0, len(f.locations))
	for _, l := range f.locations {
		locations = append(locations, l)
	}
	return locations, nil
}

func (f *fakeLocationService) Create(_ context.Context, l location.Location) (location.Location, error) {
	if f.err != nil {
		return location.Location{}, f.err
	}
	l.ID = uuid.New()
	l.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.UpdatedAt = l.CreatedAt
	f.locations[l.ID] = l
	return l, nil
}

func (f *fakeLocationService) Update(_ context.Context, l location.Location) (location.Location, error) {
	if f.err != nil {
		return location.Location{}, f.err
	}
	if _, ok := f.locations[l.ID]; !ok {
		return location.Location{}, apperror.NotFound("location not found")
	}
	f.locations[l.ID] = l
	return l, nil
}

func (f *fakeLocationService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.locations[id]; !ok {
		return apperror.NotFound("location not found")
	}
	delete(f.locations, id)
	return nil
}

// newTestRouter mounts a LocationHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeLocationService) http.Handler {
	handler := httpapi.NewLocationHandler(svc)

	r := chi.NewRouter()
	r.Post("/locations", handler.Create)
	r.Get("/locations", handler.List)
	r.Get("/locations/{id}", handler.Get)
	r.Put("/locations/{id}", handler.Update)
	r.Delete("/locations/{id}", handler.Delete)
	return r
}

const validBody = `{"customer_id":"11111111-1111-1111-1111-111111111111","name":"Main Service Address","type":"Service","status":"Active"}`

func TestLocationHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeLocationService())

	req := httptest.NewRequest(http.MethodPost, "/locations", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer_id"`
		Name       string `json:"name"`
		Type       string `json:"type"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.CustomerID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("customer_id = %q, want %q", body.CustomerID, "11111111-1111-1111-1111-111111111111")
	}
	if body.Name != "Main Service Address" || body.Type != "Service" || body.Status != "Active" {
		t.Errorf("body = %+v, want Name=Main Service Address Type=Service Status=Active", body)
	}
}

func TestLocationHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeLocationService())

	req := httptest.NewRequest(http.MethodPost, "/locations", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLocationHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeLocationService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/locations", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestLocationHandlerCreatePropagatesConflictOnUnknownCustomer(t *testing.T) {
	svc := newFakeLocationService()
	svc.err = apperror.Conflict("create location: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/locations", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestLocationHandlerList(t *testing.T) {
	a := location.Location{ID: uuid.New(), CustomerID: uuid.New(), Name: "A", Type: location.LocationTypeService, Status: location.LocationStatusActive}
	b := location.Location{ID: uuid.New(), CustomerID: uuid.New(), Name: "B", Type: location.LocationTypeOffice, Status: location.LocationStatusActive}
	router := newTestRouter(newFakeLocationService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Locations []struct {
			ID string `json:"id"`
		} `json:"locations"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Locations) != 2 {
		t.Fatalf("len(locations) = %d, want 2", len(body.Locations))
	}
}

func TestLocationHandlerGet(t *testing.T) {
	l := location.Location{ID: uuid.New(), CustomerID: uuid.New(), Name: "Main Service Address", Type: location.LocationTypeService, Status: location.LocationStatusActive}
	router := newTestRouter(newFakeLocationService(l))

	req := httptest.NewRequest(http.MethodGet, "/locations/"+l.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestLocationHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeLocationService())

	req := httptest.NewRequest(http.MethodGet, "/locations/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLocationHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeLocationService())

	req := httptest.NewRequest(http.MethodGet, "/locations/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLocationHandlerUpdate(t *testing.T) {
	l := location.Location{ID: uuid.New(), CustomerID: uuid.New(), Name: "Old Name", Type: location.LocationTypeService, Status: location.LocationStatusActive}
	router := newTestRouter(newFakeLocationService(l))

	req := httptest.NewRequest(http.MethodPut, "/locations/"+l.ID.String(),
		strings.NewReader(`{"customer_id":"`+l.CustomerID.String()+`","name":"New Name","type":"Billing","status":"Inactive"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" || body.Type != "Billing" || body.Status != "Inactive" {
		t.Errorf("body = %+v, want Name=New Name Type=Billing Status=Inactive", body)
	}
}

func TestLocationHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeLocationService())

	req := httptest.NewRequest(http.MethodPut, "/locations/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLocationHandlerDelete(t *testing.T) {
	l := location.Location{ID: uuid.New(), CustomerID: uuid.New(), Name: "Temporary", Type: location.LocationTypeService, Status: location.LocationStatusActive}
	router := newTestRouter(newFakeLocationService(l))

	req := httptest.NewRequest(http.MethodDelete, "/locations/"+l.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestLocationHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeLocationService())

	req := httptest.NewRequest(http.MethodDelete, "/locations/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
