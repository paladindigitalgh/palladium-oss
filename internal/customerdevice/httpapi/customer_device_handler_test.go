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

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeDeviceGetter is the seam httpapi.CustomerDeviceHandler uses to
// resolve the attached Device's Name for its Event message.
type fakeDeviceGetter struct {
	devices map[uuid.UUID]inventory.Device
}

func (f *fakeDeviceGetter) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	d, ok := f.devices[id]
	if !ok {
		return inventory.Device{}, apperror.NotFound("device not found")
	}
	return d, nil
}

// fakeCustomerGetter is the seam httpapi.CustomerDeviceHandler uses to
// resolve the Customer's Name for its Event message.
type fakeCustomerGetter struct {
	customers map[uuid.UUID]customer.Customer
}

func (f *fakeCustomerGetter) Get(_ context.Context, id uuid.UUID) (customer.Customer, error) {
	c, ok := f.customers[id]
	if !ok {
		return customer.Customer{}, apperror.NotFound("customer not found")
	}
	return c, nil
}

// fakeEventRecorder is the seam httpapi.CustomerDeviceHandler uses to
// record an Event after a successful attach (Create) or detach (an
// Active-true -> Active-false Update). Records every Event it's given
// so tests can assert on the exact message written.
type fakeEventRecorder struct {
	created []event.Event
}

func (f *fakeEventRecorder) Create(_ context.Context, e event.Event) (event.Event, error) {
	f.created = append(f.created, e)
	return e, nil
}

var (
	validCustomerID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	validDeviceID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
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
// own newTestRouter. Returns the fakeEventRecorder alongside the router
// so tests that care about the Event written on attach/detach can
// inspect it; most tests discard it. The seeded *Getters always resolve
// validDeviceID -> "ONT-Main-01" and validCustomerID -> "Acme Corp".
func newTestRouter(svc *fakeCustomerDeviceService) (http.Handler, *fakeEventRecorder) {
	devices := &fakeDeviceGetter{devices: map[uuid.UUID]inventory.Device{
		validDeviceID: {Metadata: inventory.Metadata{ID: validDeviceID, Name: "ONT-Main-01"}},
	}}
	customers := &fakeCustomerGetter{customers: map[uuid.UUID]customer.Customer{
		validCustomerID: {ID: validCustomerID, Name: "Acme Corp"},
	}}
	events := &fakeEventRecorder{}
	handler := httpapi.NewCustomerDeviceHandler(svc, devices, customers, events)

	r := chi.NewRouter()
	r.Post("/customer-devices", handler.Create)
	r.Get("/customer-devices", handler.List)
	r.Get("/customer-devices/{id}", handler.Get)
	r.Put("/customer-devices/{id}", handler.Update)
	return r, events
}

const validBody = `{"customer_id":"11111111-1111-1111-1111-111111111111","device_id":"22222222-2222-2222-2222-222222222222"}`

func TestCustomerDeviceHandlerCreate(t *testing.T) {
	router, events := newTestRouter(newFakeCustomerDeviceService())

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

	if len(events.created) != 1 {
		t.Fatalf("len(events.created) = %d, want 1", len(events.created))
	}
	got := events.created[0]
	if got.EntityType != "customer_device" || got.EntityID.String() != body.ID {
		t.Errorf("event EntityType/EntityID = %q/%v, want \"customer_device\"/%s", got.EntityType, got.EntityID, body.ID)
	}
	if got.Type != "customer_device.attached" {
		t.Errorf("event Type = %q, want %q", got.Type, "customer_device.attached")
	}
	if got.Message != "Attached device ONT-Main-01 to Acme Corp" {
		t.Errorf("event Message = %q, want %q", got.Message, "Attached device ONT-Main-01 to Acme Corp")
	}
}

func TestCustomerDeviceHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router, _ := newTestRouter(newFakeCustomerDeviceService())

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
	router, _ := newTestRouter(svc)

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
	router, _ := newTestRouter(svc)

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
	router, _ := newTestRouter(newFakeCustomerDeviceService(a, b))

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
	router, _ := newTestRouter(newFakeCustomerDeviceService(record))

	req := httptest.NewRequest(http.MethodGet, "/customer-devices/"+record.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestCustomerDeviceHandlerGetNotFound(t *testing.T) {
	router, _ := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodGet, "/customer-devices/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCustomerDeviceHandlerGetRejectsMalformedID(t *testing.T) {
	router, _ := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodGet, "/customer-devices/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestCustomerDeviceHandlerUpdate is a plain edit (LocationID only, the
// record stays Active() before and after) -- it must write no Event,
// since the attachment relationship itself never changed. See
// TestCustomerDeviceHandlerUpdateDetach for the actual detach
// transition.
func TestCustomerDeviceHandlerUpdate(t *testing.T) {
	record := customerdevice.CustomerDevice{ID: uuid.New(), CustomerID: validCustomerID, DeviceID: validDeviceID}
	router, events := newTestRouter(newFakeCustomerDeviceService(record))

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

	if len(events.created) != 0 {
		t.Errorf("len(events.created) = %d, want 0 (a plain edit records no Event)", len(events.created))
	}
}

// TestCustomerDeviceHandlerUpdateDetach is the actual detach transition
// (DetachedAt goes from unset to set, Active() true -> false) -- the one
// Update call that does write a "customer_device.detached" Event.
func TestCustomerDeviceHandlerUpdateDetach(t *testing.T) {
	record := customerdevice.CustomerDevice{ID: uuid.New(), CustomerID: validCustomerID, DeviceID: validDeviceID}
	router, events := newTestRouter(newFakeCustomerDeviceService(record))

	detachedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodPut, "/customer-devices/"+record.ID.String(), strings.NewReader(
		`{"customer_id":"`+record.CustomerID.String()+`","device_id":"`+record.DeviceID.String()+`","detached_at":"`+detachedAt+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if len(events.created) != 1 {
		t.Fatalf("len(events.created) = %d, want 1", len(events.created))
	}
	got := events.created[0]
	if got.EntityType != "customer_device" || got.EntityID != record.ID {
		t.Errorf("event EntityType/EntityID = %q/%v, want \"customer_device\"/%v", got.EntityType, got.EntityID, record.ID)
	}
	if got.Type != "customer_device.detached" {
		t.Errorf("event Type = %q, want %q", got.Type, "customer_device.detached")
	}
	if got.Message != "Detached device ONT-Main-01 from Acme Corp" {
		t.Errorf("event Message = %q, want %q", got.Message, "Detached device ONT-Main-01 from Acme Corp")
	}
}

func TestCustomerDeviceHandlerUpdateNotFound(t *testing.T) {
	router, _ := newTestRouter(newFakeCustomerDeviceService())

	req := httptest.NewRequest(http.MethodPut, "/customer-devices/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
