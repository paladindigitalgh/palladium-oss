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

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAccessInterfaceService is the seam
// httpapi.AccessInterfaceHandler depends on (see its unexported
// accessInterfaceService interface in access_interface_handler.go). It
// lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, routing, error translation — without a real service,
// repository, or database; internal/accessinterface/service and
// internal/accessinterface/postgres each have their own tests for the
// layers below this one.
type fakeAccessInterfaceService struct {
	interfaces map[uuid.UUID]accessinterface.AccessInterface
	err        error // if set, every method returns this error instead
}

func newFakeAccessInterfaceService(interfaces ...accessinterface.AccessInterface) *fakeAccessInterfaceService {
	f := &fakeAccessInterfaceService{interfaces: make(map[uuid.UUID]accessinterface.AccessInterface)}
	for _, a := range interfaces {
		f.interfaces[a.ID] = a
	}
	return f
}

func (f *fakeAccessInterfaceService) Get(_ context.Context, id uuid.UUID) (accessinterface.AccessInterface, error) {
	if f.err != nil {
		return accessinterface.AccessInterface{}, f.err
	}
	a, ok := f.interfaces[id]
	if !ok {
		return accessinterface.AccessInterface{}, apperror.NotFound("access interface not found")
	}
	return a, nil
}

func (f *fakeAccessInterfaceService) List(context.Context) ([]accessinterface.AccessInterface, error) {
	if f.err != nil {
		return nil, f.err
	}
	interfaces := make([]accessinterface.AccessInterface, 0, len(f.interfaces))
	for _, a := range f.interfaces {
		interfaces = append(interfaces, a)
	}
	return interfaces, nil
}

func (f *fakeAccessInterfaceService) Create(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	if f.err != nil {
		return accessinterface.AccessInterface{}, f.err
	}
	a.ID = uuid.New()
	a.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.UpdatedAt = a.CreatedAt
	f.interfaces[a.ID] = a
	return a, nil
}

func (f *fakeAccessInterfaceService) Update(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	if f.err != nil {
		return accessinterface.AccessInterface{}, f.err
	}
	if _, ok := f.interfaces[a.ID]; !ok {
		return accessinterface.AccessInterface{}, apperror.NotFound("access interface not found")
	}
	f.interfaces[a.ID] = a
	return a, nil
}

func (f *fakeAccessInterfaceService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.interfaces[id]; !ok {
		return apperror.NotFound("access interface not found")
	}
	delete(f.interfaces, id)
	return nil
}

// newTestRouter mounts an AccessInterfaceHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeAccessInterfaceService) http.Handler {
	handler := httpapi.NewAccessInterfaceHandler(svc)

	r := chi.NewRouter()
	r.Post("/access-interfaces", handler.Create)
	r.Get("/access-interfaces", handler.List)
	r.Get("/access-interfaces/{id}", handler.Get)
	r.Put("/access-interfaces/{id}", handler.Update)
	r.Delete("/access-interfaces/{id}", handler.Delete)
	return r
}

const validBody = `{"pon_port_id":"11111111-1111-1111-1111-111111111111","technology":"GPON","name":"gpon-0/1/1","status":"Active"}`

func TestAccessInterfaceHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeAccessInterfaceService())

	req := httptest.NewRequest(http.MethodPost, "/access-interfaces", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		PONPortID  string `json:"pon_port_id"`
		Technology string `json:"technology"`
		Name       string `json:"name"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.PONPortID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("pon_port_id = %q, want %q", body.PONPortID, "11111111-1111-1111-1111-111111111111")
	}
	if body.Technology != "GPON" {
		t.Errorf("technology = %q, want %q", body.Technology, "GPON")
	}
	if body.Name != "gpon-0/1/1" {
		t.Errorf("name = %q, want %q", body.Name, "gpon-0/1/1")
	}
	if body.Status != "Active" {
		t.Errorf("status = %q, want %q", body.Status, "Active")
	}
}

func TestAccessInterfaceHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeAccessInterfaceService())

	req := httptest.NewRequest(http.MethodPost, "/access-interfaces", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAccessInterfaceHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeAccessInterfaceService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/access-interfaces", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAccessInterfaceHandlerCreatePropagatesConflictOnUnknownPONPort(t *testing.T) {
	svc := newFakeAccessInterfaceService()
	svc.err = apperror.Conflict("create access interface: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/access-interfaces", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestAccessInterfaceHandlerList(t *testing.T) {
	a := accessinterface.AccessInterface{ID: uuid.New(), PONPortID: uuid.New(), Technology: accessinterface.TechnologyGPON, Name: "Alpha", Status: accessinterface.StatusActive}
	b := accessinterface.AccessInterface{ID: uuid.New(), PONPortID: uuid.New(), Technology: accessinterface.TechnologyGPON, Name: "Beta", Status: accessinterface.StatusActive}
	router := newTestRouter(newFakeAccessInterfaceService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/access-interfaces", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		AccessInterfaces []struct {
			ID string `json:"id"`
		} `json:"access_interfaces"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.AccessInterfaces) != 2 {
		t.Fatalf("len(access_interfaces) = %d, want 2", len(body.AccessInterfaces))
	}
}

func TestAccessInterfaceHandlerGet(t *testing.T) {
	a := accessinterface.AccessInterface{ID: uuid.New(), PONPortID: uuid.New(), Technology: accessinterface.TechnologyGPON, Name: "Alpha", Status: accessinterface.StatusActive}
	router := newTestRouter(newFakeAccessInterfaceService(a))

	req := httptest.NewRequest(http.MethodGet, "/access-interfaces/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestAccessInterfaceHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessInterfaceService())

	req := httptest.NewRequest(http.MethodGet, "/access-interfaces/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAccessInterfaceHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeAccessInterfaceService())

	req := httptest.NewRequest(http.MethodGet, "/access-interfaces/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAccessInterfaceHandlerUpdate(t *testing.T) {
	a := accessinterface.AccessInterface{ID: uuid.New(), PONPortID: uuid.New(), Technology: accessinterface.TechnologyGPON, Name: "Alpha", Status: accessinterface.StatusActive}
	router := newTestRouter(newFakeAccessInterfaceService(a))

	req := httptest.NewRequest(http.MethodPut, "/access-interfaces/"+a.ID.String(),
		strings.NewReader(`{"pon_port_id":"`+a.PONPortID.String()+`","technology":"GPON","name":"Beta","status":"Disabled"}`))
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
	if body.Status != "Disabled" {
		t.Errorf("status = %q, want %q", body.Status, "Disabled")
	}
}

func TestAccessInterfaceHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessInterfaceService())

	req := httptest.NewRequest(http.MethodPut, "/access-interfaces/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAccessInterfaceHandlerDelete(t *testing.T) {
	a := accessinterface.AccessInterface{ID: uuid.New(), PONPortID: uuid.New(), Technology: accessinterface.TechnologyGPON, Name: "Alpha", Status: accessinterface.StatusActive}
	router := newTestRouter(newFakeAccessInterfaceService(a))

	req := httptest.NewRequest(http.MethodDelete, "/access-interfaces/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestAccessInterfaceHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessInterfaceService())

	req := httptest.NewRequest(http.MethodDelete, "/access-interfaces/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
