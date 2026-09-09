package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/httpapi"
)

// fakeAuthorizeAndCreateDeviceService is the seam
// httpapi.AuthorizeAndCreateDeviceHandler depends on, mirroring
// fakeAuthorizationService above.
type fakeAuthorizeAndCreateDeviceService struct {
	device inventory.Device
	iface  string
	err    error

	gotOLTID  uuid.UUID
	gotPort   string
	gotDevice inventory.Device
	called    bool
}

func (f *fakeAuthorizeAndCreateDeviceService) AuthorizeAndCreateDevice(_ context.Context, oltID uuid.UUID, port string, device inventory.Device) (inventory.Device, string, error) {
	f.called = true
	f.gotOLTID, f.gotPort, f.gotDevice = oltID, port, device
	if f.err != nil {
		return inventory.Device{}, "", f.err
	}
	return f.device, f.iface, nil
}

func newAuthorizeAndCreateDeviceTestRouter(svc *fakeAuthorizeAndCreateDeviceService) http.Handler {
	handler := httpapi.NewAuthorizeAndCreateDeviceHandler(svc)

	r := chi.NewRouter()
	r.Route("/provisioning/olts/{oltId}", func(r chi.Router) {
		r.Post("/authorize-and-create-device", handler.AuthorizeAndCreateDevice)
	})
	return r
}

func TestAuthorizeAndCreateDeviceEndpoint(t *testing.T) {
	oltID := uuid.New()
	deviceID := uuid.New()
	modelID := uuid.New()
	svc := &fakeAuthorizeAndCreateDeviceService{
		device: inventory.Device{
			Metadata:      inventory.Metadata{ID: deviceID, Name: "New ONU"},
			DeviceModelID: modelID,
			SerialNumber:  "ISKT2308DD88",
			Status:        inventory.DeviceStatusInstalled,
		},
		iface: "xgs/6/2",
	}
	router := newAuthorizeAndCreateDeviceTestRouter(svc)

	body := `{"port":"xgs/6","serial_number":"ISKT2308DD88","name":"New ONU","device_model_id":"` + modelID.String() + `","status":"Installed"}`
	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+oltID.String()+"/authorize-and-create-device", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if !svc.called {
		t.Fatal("AuthorizeAndCreateDevice was not called")
	}
	if svc.gotOLTID != oltID || svc.gotPort != "xgs/6" {
		t.Errorf("got oltID=%v port=%q", svc.gotOLTID, svc.gotPort)
	}
	if svc.gotDevice.SerialNumber != "ISKT2308DD88" || svc.gotDevice.DeviceModelID != modelID {
		t.Errorf("got device = %+v", svc.gotDevice)
	}

	var resp struct {
		Interface     string `json:"interface"`
		ID            string `json:"id"`
		DeviceModelID string `json:"device_model_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Interface != "xgs/6/2" {
		t.Errorf("interface = %q, want %q", resp.Interface, "xgs/6/2")
	}
	if resp.ID != deviceID.String() {
		t.Errorf("id = %q, want %q", resp.ID, deviceID.String())
	}
}

func TestAuthorizeAndCreateDeviceEndpointRejectsMissingPort(t *testing.T) {
	svc := &fakeAuthorizeAndCreateDeviceService{}
	router := newAuthorizeAndCreateDeviceTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-and-create-device",
		strings.NewReader(`{"serial_number":"ISKT2308DD88"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if svc.called {
		t.Error("service was called; it must never be reached for a missing port")
	}
}

func TestAuthorizeAndCreateDeviceEndpointRejectsMissingSerialNumber(t *testing.T) {
	svc := &fakeAuthorizeAndCreateDeviceService{}
	router := newAuthorizeAndCreateDeviceTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-and-create-device",
		strings.NewReader(`{"port":"xgs/6"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if svc.called {
		t.Error("service was called; it must never be reached for a missing serial_number")
	}
}

func TestAuthorizeAndCreateDeviceEndpointPropagatesServiceErrorKinds(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"unavailable", apperror.Unavailable("could not reach OLT", context.DeadlineExceeded), http.StatusServiceUnavailable},
		{"invalid", apperror.Invalid("device_model_id is required"), http.StatusBadRequest},
		{"conflict", apperror.Conflict("a device with this serial number already exists"), http.StatusConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeAuthorizeAndCreateDeviceService{err: tc.err}
			router := newAuthorizeAndCreateDeviceTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-and-create-device",
				strings.NewReader(`{"port":"xgs/6","serial_number":"ISKT2308DD88"}`))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}
