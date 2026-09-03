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

// fakeDeviceService is the seam httpapi.DeviceHandler depends on. See
// fakeSiteService's doc comment (site_handler_test.go) for why this
// exists instead of a real service, repository, or database.
type fakeDeviceService struct {
	devices map[uuid.UUID]inventory.Device
	err     error // if set, every method returns this error instead
}

func newFakeDeviceService(devices ...inventory.Device) *fakeDeviceService {
	f := &fakeDeviceService{devices: make(map[uuid.UUID]inventory.Device)}
	for _, d := range devices {
		f.devices[d.ID] = d
	}
	return f
}

func (f *fakeDeviceService) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	if f.err != nil {
		return inventory.Device{}, f.err
	}
	d, ok := f.devices[id]
	if !ok {
		return inventory.Device{}, apperror.NotFound("device not found")
	}
	return d, nil
}

func (f *fakeDeviceService) List(context.Context) ([]inventory.Device, error) {
	if f.err != nil {
		return nil, f.err
	}
	devices := make([]inventory.Device, 0, len(f.devices))
	for _, d := range f.devices {
		devices = append(devices, d)
	}
	return devices, nil
}

func (f *fakeDeviceService) Create(_ context.Context, device inventory.Device) (inventory.Device, error) {
	if f.err != nil {
		return inventory.Device{}, f.err
	}
	device.ID = uuid.New()
	device.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	device.UpdatedAt = device.CreatedAt
	f.devices[device.ID] = device
	return device, nil
}

func (f *fakeDeviceService) Update(_ context.Context, device inventory.Device) (inventory.Device, error) {
	if f.err != nil {
		return inventory.Device{}, f.err
	}
	if _, ok := f.devices[device.ID]; !ok {
		return inventory.Device{}, apperror.NotFound("device not found")
	}
	f.devices[device.ID] = device
	return device, nil
}

func (f *fakeDeviceService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.devices[id]; !ok {
		return apperror.NotFound("device not found")
	}
	delete(f.devices, id)
	return nil
}

// newDeviceTestRouter mounts a DeviceHandler backed by svc on a real
// chi.Router. See newTestRouter's doc comment (site_handler_test.go) for
// why.
func newDeviceTestRouter(svc *fakeDeviceService) http.Handler {
	handler := httpapi.NewDeviceHandler(svc)

	r := chi.NewRouter()
	r.Post("/devices", handler.Create)
	r.Get("/devices", handler.List)
	r.Get("/devices/{id}", handler.Get)
	r.Put("/devices/{id}", handler.Update)
	r.Delete("/devices/{id}", handler.Delete)
	return r
}

const validDeviceBody = `{"name":"ONT-Main-01","manufacturer":"Calix","model":"716GE","serial_number":"CXNK00112233","status":"InStock"}`

func TestDeviceHandlerCreate(t *testing.T) {
	router := newDeviceTestRouter(newFakeDeviceService())

	req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(validDeviceBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Manufacturer string `json:"manufacturer"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "ONT-Main-01" || body.Manufacturer != "Calix" || body.Status != "InStock" {
		t.Errorf("body = %+v, want Name=ONT-Main-01 Manufacturer=Calix Status=InStock", body)
	}
}

func TestDeviceHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newDeviceTestRouter(newFakeDeviceService())

	req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeviceHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeDeviceService()
	svc.err = apperror.Invalid("name: is required")
	router := newDeviceTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestDeviceHandlerList(t *testing.T) {
	a := inventory.Device{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}, Status: inventory.DeviceStatusInStock}
	b := inventory.Device{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}, Status: inventory.DeviceStatusInStock}
	router := newDeviceTestRouter(newFakeDeviceService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Devices []struct {
			ID string `json:"id"`
		} `json:"devices"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Devices) != 2 {
		t.Fatalf("len(devices) = %d, want 2", len(body.Devices))
	}
}

func TestDeviceHandlerGet(t *testing.T) {
	device := inventory.Device{Metadata: inventory.Metadata{ID: uuid.New(), Name: "ONT-Main-01"}, Status: inventory.DeviceStatusInStock}
	router := newDeviceTestRouter(newFakeDeviceService(device))

	req := httptest.NewRequest(http.MethodGet, "/devices/"+device.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestDeviceHandlerGetNotFound(t *testing.T) {
	router := newDeviceTestRouter(newFakeDeviceService())

	req := httptest.NewRequest(http.MethodGet, "/devices/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDeviceHandlerGetRejectsMalformedID(t *testing.T) {
	router := newDeviceTestRouter(newFakeDeviceService())

	req := httptest.NewRequest(http.MethodGet, "/devices/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeviceHandlerUpdate(t *testing.T) {
	device := inventory.Device{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}, Status: inventory.DeviceStatusInStock}
	router := newDeviceTestRouter(newFakeDeviceService(device))

	req := httptest.NewRequest(http.MethodPut, "/devices/"+device.ID.String(), strings.NewReader(
		`{"name":"New Name","manufacturer":"Calix","model":"716GE","serial_number":"CXNK00112233","status":"Installed"}`))
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

func TestDeviceHandlerUpdateNotFound(t *testing.T) {
	router := newDeviceTestRouter(newFakeDeviceService())

	req := httptest.NewRequest(http.MethodPut, "/devices/"+uuid.New().String(), strings.NewReader(validDeviceBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDeviceHandlerDelete(t *testing.T) {
	device := inventory.Device{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}, Status: inventory.DeviceStatusInStock}
	router := newDeviceTestRouter(newFakeDeviceService(device))

	req := httptest.NewRequest(http.MethodDelete, "/devices/"+device.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestDeviceHandlerDeleteNotFound(t *testing.T) {
	router := newDeviceTestRouter(newFakeDeviceService())

	req := httptest.NewRequest(http.MethodDelete, "/devices/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
