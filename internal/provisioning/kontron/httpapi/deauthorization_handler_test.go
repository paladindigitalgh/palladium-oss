package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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

// newDeauthorizationTestRouter mounts a DeauthorizationHandler on a real
// chi.Router, mirroring how internal/server/router.go mounts it in
// production (minus auth/authz).
func newDeauthorizationTestRouter(svc *fakeDeauthorizationService) http.Handler {
	handler := httpapi.NewDeauthorizationHandler(svc)

	r := chi.NewRouter()
	r.Route("/provisioning/devices/{deviceId}", func(r chi.Router) {
		r.Post("/deauthorize-onu", handler.DeauthorizeONU)
	})
	return r
}

func TestDeauthorizeONUEndpoint(t *testing.T) {
	deviceID := uuid.New()
	svc := &fakeDeauthorizationService{iface: "xgs/6/2"}
	router := newDeauthorizationTestRouter(svc)

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
}

func TestDeauthorizeONUEndpointRejectsInvalidDeviceID(t *testing.T) {
	svc := &fakeDeauthorizationService{}
	router := newDeauthorizationTestRouter(svc)

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
			router := newDeauthorizationTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/provisioning/devices/"+uuid.New().String()+"/deauthorize-onu", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}
