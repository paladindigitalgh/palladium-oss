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
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/httpapi"
)

// fakeServiceEquipmentService is the seam
// httpapi.ServiceEquipmentHandler depends on (see its unexported
// serviceEquipmentService interface in service_equipment_handler.go). It
// lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, routing, error translation — without a real service,
// repository, or database; internal/serviceequipment/service and
// internal/serviceequipment/postgres each have their own tests for the
// layers below this one, including internal/serviceequipment/service's
// own tests for the active-assignment-uniqueness rule itself.
type fakeServiceEquipmentService struct {
	equipment map[uuid.UUID]serviceequipment.ServiceEquipment
	err       error // if set, every method returns this error instead
}

func newFakeServiceEquipmentService(equipment ...serviceequipment.ServiceEquipment) *fakeServiceEquipmentService {
	f := &fakeServiceEquipmentService{equipment: make(map[uuid.UUID]serviceequipment.ServiceEquipment)}
	for _, e := range equipment {
		f.equipment[e.ID] = e
	}
	return f
}

func (f *fakeServiceEquipmentService) Get(_ context.Context, id uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	if f.err != nil {
		return serviceequipment.ServiceEquipment{}, f.err
	}
	e, ok := f.equipment[id]
	if !ok {
		return serviceequipment.ServiceEquipment{}, apperror.NotFound("service equipment not found")
	}
	return e, nil
}

func (f *fakeServiceEquipmentService) List(context.Context) ([]serviceequipment.ServiceEquipment, error) {
	if f.err != nil {
		return nil, f.err
	}
	equipment := make([]serviceequipment.ServiceEquipment, 0, len(f.equipment))
	for _, e := range f.equipment {
		equipment = append(equipment, e)
	}
	return equipment, nil
}

func (f *fakeServiceEquipmentService) Create(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	if f.err != nil {
		return serviceequipment.ServiceEquipment{}, f.err
	}
	e.ID = uuid.New()
	e.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e.UpdatedAt = e.CreatedAt
	f.equipment[e.ID] = e
	return e, nil
}

func (f *fakeServiceEquipmentService) Update(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	if f.err != nil {
		return serviceequipment.ServiceEquipment{}, f.err
	}
	if _, ok := f.equipment[e.ID]; !ok {
		return serviceequipment.ServiceEquipment{}, apperror.NotFound("service equipment not found")
	}
	f.equipment[e.ID] = e
	return e, nil
}

func (f *fakeServiceEquipmentService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.equipment[id]; !ok {
		return apperror.NotFound("service equipment not found")
	}
	delete(f.equipment, id)
	return nil
}

// newTestRouter mounts a ServiceEquipmentHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeServiceEquipmentService) http.Handler {
	handler := httpapi.NewServiceEquipmentHandler(svc)

	r := chi.NewRouter()
	r.Post("/service-equipment", handler.Create)
	r.Get("/service-equipment", handler.List)
	r.Get("/service-equipment/{id}", handler.Get)
	r.Put("/service-equipment/{id}", handler.Update)
	r.Delete("/service-equipment/{id}", handler.Delete)
	return r
}

const validBody = `{"service_id":"11111111-1111-1111-1111-111111111111","device_id":"22222222-2222-2222-2222-222222222222","role":"ONU"}`

func TestServiceEquipmentHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeServiceEquipmentService())

	req := httptest.NewRequest(http.MethodPost, "/service-equipment", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID        string `json:"id"`
		ServiceID string `json:"service_id"`
		DeviceID  string `json:"device_id"`
		Role      string `json:"role"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.ServiceID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("service_id = %q, want %q", body.ServiceID, "11111111-1111-1111-1111-111111111111")
	}
	if body.DeviceID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("device_id = %q, want %q", body.DeviceID, "22222222-2222-2222-2222-222222222222")
	}
	if body.Role != "ONU" {
		t.Errorf("role = %q, want %q", body.Role, "ONU")
	}
}

func TestServiceEquipmentHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeServiceEquipmentService())

	req := httptest.NewRequest(http.MethodPost, "/service-equipment", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceEquipmentHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeServiceEquipmentService()
	svc.err = apperror.Invalid("role: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/service-equipment", strings.NewReader(`{"role":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestServiceEquipmentHandlerCreatePropagatesConflictOnUnknownServiceOrDevice(t *testing.T) {
	svc := newFakeServiceEquipmentService()
	svc.err = apperror.Conflict("create service equipment: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/service-equipment", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

// TestServiceEquipmentHandlerCreatePropagatesConflictOnAlreadyActiveDevice
// proves the active-assignment-uniqueness Conflict from
// ServiceEquipmentService (goal 2) reaches the client as 409, the same as
// any other Conflict — this handler has no special-case knowledge of that
// rule, it just translates whatever error the service layer returns.
func TestServiceEquipmentHandlerCreatePropagatesConflictOnAlreadyActiveDevice(t *testing.T) {
	svc := newFakeServiceEquipmentService()
	svc.err = apperror.Conflict("device already has an active service equipment assignment")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/service-equipment", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestServiceEquipmentHandlerList(t *testing.T) {
	a := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: uuid.New(), DeviceID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	b := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: uuid.New(), DeviceID: uuid.New(), Role: serviceequipment.EquipmentRoleRouter}
	router := newTestRouter(newFakeServiceEquipmentService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/service-equipment", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		ServiceEquipment []struct {
			ID string `json:"id"`
		} `json:"service_equipment"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.ServiceEquipment) != 2 {
		t.Fatalf("len(service_equipment) = %d, want 2", len(body.ServiceEquipment))
	}
}

func TestServiceEquipmentHandlerGet(t *testing.T) {
	e := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: uuid.New(), DeviceID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	router := newTestRouter(newFakeServiceEquipmentService(e))

	req := httptest.NewRequest(http.MethodGet, "/service-equipment/"+e.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestServiceEquipmentHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceEquipmentService())

	req := httptest.NewRequest(http.MethodGet, "/service-equipment/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServiceEquipmentHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeServiceEquipmentService())

	req := httptest.NewRequest(http.MethodGet, "/service-equipment/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServiceEquipmentHandlerUpdate(t *testing.T) {
	e := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: uuid.New(), DeviceID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	router := newTestRouter(newFakeServiceEquipmentService(e))

	req := httptest.NewRequest(http.MethodPut, "/service-equipment/"+e.ID.String(),
		strings.NewReader(`{"service_id":"`+e.ServiceID.String()+`","device_id":"`+e.DeviceID.String()+`","role":"Router"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Role != "Router" {
		t.Errorf("role = %q, want %q", body.Role, "Router")
	}
}

func TestServiceEquipmentHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceEquipmentService())

	req := httptest.NewRequest(http.MethodPut, "/service-equipment/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServiceEquipmentHandlerDelete(t *testing.T) {
	e := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: uuid.New(), DeviceID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	router := newTestRouter(newFakeServiceEquipmentService(e))

	req := httptest.NewRequest(http.MethodDelete, "/service-equipment/"+e.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestServiceEquipmentHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeServiceEquipmentService())

	req := httptest.NewRequest(http.MethodDelete, "/service-equipment/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
