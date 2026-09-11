package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/httpapi"
)

// fakeDeauthorizationService is the seam
// httpapi.DeauthorizationHandler depends on (see its unexported
// deauthorizationService interface) — the same technique
// fakeAuthorizationService uses for AuthorizationHandler's own tests.
type fakeDeauthorizationService struct {
	iface       string
	err         error
	gotDeviceID uuid.UUID
	called      bool
}

func (f *fakeDeauthorizationService) DeauthorizeONU(_ context.Context, deviceID uuid.UUID) (string, error) {
	f.called = true
	f.gotDeviceID = deviceID
	if f.err != nil {
		return "", f.err
	}
	return f.iface, nil
}

// fakeDeviceGetter is the seam DeauthorizationHandler uses to resolve
// the retired Device's Name for its Event message.
type fakeDeviceGetter struct {
	device inventory.Device
	err    error
}

func (f *fakeDeviceGetter) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	if f.err != nil {
		return inventory.Device{}, f.err
	}
	f.device.ID = id
	return f.device, nil
}

// fakeEventRecorder is the seam DeauthorizationHandler uses to record a
// "device.retired" Event on success. Records every Event it's given so
// tests can assert on the exact message written.
type fakeEventRecorder struct {
	created []event.Event
}

func (f *fakeEventRecorder) Create(_ context.Context, e event.Event) (event.Event, error) {
	f.created = append(f.created, e)
	return e, nil
}

// newDeauthorizationTestRouter mounts a DeauthorizationHandler on a real
// chi.Router, mirroring how internal/server/router.go mounts it in
// production (minus auth/authz). Returns the fakeDeviceGetter/
// fakeEventRecorder alongside the router so tests that care about the
// resolved Device or the Event written can inspect them; most tests
// discard both.
func newDeauthorizationTestRouter(svc *fakeDeauthorizationService) (http.Handler, *fakeDeviceGetter, *fakeEventRecorder) {
	devices := &fakeDeviceGetter{device: inventory.Device{Metadata: inventory.Metadata{Name: "ONT-Main-01"}}}
	events := &fakeEventRecorder{}
	handler := httpapi.NewDeauthorizationHandler(svc, devices, events)

	r := chi.NewRouter()
	r.Route("/provisioning/devices/{deviceId}", func(r chi.Router) {
		r.Post("/deauthorize-onu", handler.DeauthorizeONU)
	})
	return r, devices, events
}

func TestDeauthorizeONUEndpoint(t *testing.T) {
	deviceID := uuid.New()
	svc := &fakeDeauthorizationService{iface: "xgs/6/2"}
	router, _, events := newDeauthorizationTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/devices/"+deviceID.String()+"/deauthorize-onu", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !svc.called {
		t.Fatal("DeauthorizeONU was not called")
	}
	if svc.gotDeviceID != deviceID {
		t.Errorf("gotDeviceID = %v, want %v", svc.gotDeviceID, deviceID)
	}

	var body struct {
		Interface string `json:"interface"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Interface != "xgs/6/2" {
		t.Errorf("interface = %q, want %q", body.Interface, "xgs/6/2")
	}

	if len(events.created) != 1 {
		t.Fatalf("len(events.created) = %d, want 1", len(events.created))
	}
	got := events.created[0]
	if got.EntityType != "device" || got.EntityID != deviceID {
		t.Errorf("event EntityType/EntityID = %q/%v, want \"device\"/%v", got.EntityType, got.EntityID, deviceID)
	}
	if got.Type != "device.retired" {
		t.Errorf("event Type = %q, want %q", got.Type, "device.retired")
	}
	if got.Message != "Retired device ONT-Main-01" {
		t.Errorf("event Message = %q, want %q", got.Message, "Retired device ONT-Main-01")
	}
}

func TestDeauthorizeONUEndpointRejectsInvalidDeviceID(t *testing.T) {
	svc := &fakeDeauthorizationService{}
	router, _, _ := newDeauthorizationTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/devices/not-a-uuid/deauthorize-onu", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if svc.called {
		t.Error("service was called; it must never be reached for an invalid deviceId")
	}
}

// TestDeauthorizeONUPropagatesServiceErrorKinds proves each apperror.Kind
// DeauthorizationService can return maps to the HTTP status
// httpx.WriteError already establishes for it, mirroring
// AuthorizationHandler's own TestPropagatesServiceErrorKinds.
func TestDeauthorizeONUPropagatesServiceErrorKinds(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"unavailable", apperror.Unavailable("could not reach OLT", context.DeadlineExceeded), http.StatusServiceUnavailable},
		{"invalid", apperror.Invalid("device is not an ONU or ONT"), http.StatusBadRequest},
		{"not_found", apperror.NotFound("no active equipment for this device"), http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeDeauthorizationService{err: tc.err}
			router, _, _ := newDeauthorizationTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/provisioning/devices/"+uuid.New().String()+"/deauthorize-onu", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}
