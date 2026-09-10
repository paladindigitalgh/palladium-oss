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

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeCustomerDeviceService is the seam httpapi.CustomerDeviceHandler
// depends on, mirroring
// internal/serviceequipment/httpapi/service_equipment_handler_test.go's
// fakeServiceEquipmentService exactly.
type fakeCustomerDeviceService struct {
	records map[uuid.UUID]customerdevice.CustomerDevice
	err     error // if set, every method returns this error instead
}

func newFakeCustomerDeviceService(records ...customerdevice.CustomerDevice) *fakeCustomerDeviceService {
	f := &fakeCustomerDeviceService{records: make(map[uuid.UUID]customerdevice.CustomerDevice)}
	for _, r := range records {
		f.records[r.ID] = r
	}
	return f
}

func (f *fakeCustomerDeviceService) Get(_ context.Context, id uuid.UUID) (customerdevice.CustomerDevice, error) {
	if f.err != nil {
		return customerdevice.CustomerDevice{}, f.err
	}
	r, ok := f.records[id]
	if !ok {
		return customerdevice.CustomerDevice{}, apperror.NotFound("customer device not found")
	}
	return r, nil
}

func (f *fakeCustomerDeviceService) List(context.Context) ([]customerdevice.CustomerDevice, error) {
	if f.err != nil {
		return nil, f.err
	}
	records := make([]customerdevice.CustomerDevice, 0, len(f.records))
	for _, r := range f.records {
		records = append(records, r)
	}
	return records, nil
}

func (f *fakeCustomerDeviceService) Create(_ context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	if f.err != nil {
		return customerdevice.CustomerDevice{}, f.err
	}
	cd.ID = uuid.New()
	cd.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cd.UpdatedAt = cd.CreatedAt
	f.records[cd.ID] = cd
	return cd, nil
}

func (f *fakeCustomerDeviceService) Update(_ context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	if f.err != nil {
		return customerdevice.CustomerDevice{}, f.err
	}
	if _, ok := f.records[cd.ID]; !ok {
		return customerdevice.CustomerDevice{}, apperror.NotFound("customer device not found")
	}
	f.records[cd.ID] = cd
	return cd, nil
}

// newTestRouter mounts a CustomerDeviceHandler backed by svc on a real
// chi.Router, mirroring
// internal/serviceequipment/httpapi/service_equipment_handler_test.go's
// own newTestRouter.
func newTestRouter(svc *fakeCustomerDeviceService) http.Handler {
	handler := httpapi.NewCustomerDeviceHandler(svc)

	r := chi.NewRouter()
	r.Post("/customer-devices", handler.Create)
	r.Get("/customer-devices", handler.List)
	r.Get("/customer-devices/{id}", handler.Get)
	r.Put("/customer-devices/{id}", handler.Update)
	return r
}

const validBody = `{"customer_id":"11111111-1111-1111-1111-111111111111","device_id":"22222222-2222-2222-2222-222222222222"}`

func TestCustomerDeviceHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodPost, "/customer-devices", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer_id"`
		DeviceID   string `json:"device_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.CustomerID != "11111111-1111-1111-1111-111111111111" || body.DeviceID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("body = %+v, want CustomerID/DeviceID matching request", body)
	}
}

func TestCustomerDeviceHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodPost, "/customer-devices", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCustomerDeviceHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeCustomerDeviceService()
	svc.err = apperror.Invalid("customer_id: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/customer-devices", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCustomerDeviceHandlerCreatePropagatesConflict(t *testing.T) {
	svc := newFakeCustomerDeviceService()
	svc.err = apperror.Conflict("device already attached to another customer")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/customer-devices", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestCustomerDeviceHandlerList(t *testing.T) {
	a := customerdevice.CustomerDevice{ID: uuid.New(), CustomerID: uuid.New(), DeviceID: uuid.New()}
	b := customerdevice.CustomerDevice{ID: uuid.New(), CustomerID: uuid.New(), DeviceID: uuid.New()}
	router := newTestRouter(newFakeCustomerDeviceService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/customer-devices", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		CustomerDevices []struct {
			ID string `json:"id"`
		} `json:"customer_devices"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.CustomerDevices) != 2 {
		t.Fatalf("len(customer_devices) = %d, want 2", len(body.CustomerDevices))
	}
}

func TestCustomerDeviceHandlerGet(t *testing.T) {
	record := customerdevice.CustomerDevice{ID: uuid.New(), CustomerID: uuid.New(), DeviceID: uuid.New()}
	router := newTestRouter(newFakeCustomerDeviceService(record))

	req := httptest.NewRequest(http.MethodGet, "/customer-devices/"+record.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestCustomerDeviceHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodGet, "/customer-devices/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCustomerDeviceHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodGet, "/customer-devices/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCustomerDeviceHandlerUpdate(t *testing.T) {
	record := customerdevice.CustomerDevice{ID: uuid.New(), CustomerID: uuid.New(), DeviceID: uuid.New()}
	router := newTestRouter(newFakeCustomerDeviceService(record))

	locationID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/customer-devices/"+record.ID.String(), strings.NewReader(
		`{"customer_id":"`+record.CustomerID.String()+`","device_id":"`+record.DeviceID.String()+`","location_id":"`+locationID.String()+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		LocationID string `json:"location_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.LocationID != locationID.String() {
		t.Errorf("LocationID = %q, want %q", body.LocationID, locationID.String())
	}
}

func TestCustomerDeviceHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodPut, "/customer-devices/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
