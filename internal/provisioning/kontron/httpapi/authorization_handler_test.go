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

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/httpapi"
)

// fakeAuthorizationService is the seam httpapi.AuthorizationHandler
// depends on (see its unexported authorizationService interface) — the
// same technique internal/diagnostics/kontron/httpapi's fakeKontronService
// uses for that package's own handler tests.
type fakeAuthorizationService struct {
	iface     string
	err       error
	gotOLTID  uuid.UUID
	gotPort   string
	gotSerial string
	called    bool
}

func (f *fakeAuthorizationService) AuthorizeONU(_ context.Context, oltID uuid.UUID, port, serialNumber string) (string, error) {
	f.called = true
	f.gotOLTID, f.gotPort, f.gotSerial = oltID, port, serialNumber
	if f.err != nil {
		return "", f.err
	}
	return f.iface, nil
}

// newTestRouter mounts an AuthorizationHandler on a real chi.Router,
// mirroring how internal/server/router.go mounts it in production
// (minus auth/authz — see authenticated_test.go for that).
func newTestRouter(svc *fakeAuthorizationService) http.Handler {
	handler := httpapi.NewAuthorizationHandler(svc)

	r := chi.NewRouter()
	r.Route("/provisioning/olts/{oltId}", func(r chi.Router) {
		r.Post("/authorize-onu", handler.AuthorizeONU)
	})
	return r
}

func TestAuthorizeONUEndpoint(t *testing.T) {
	oltID := uuid.New()
	svc := &fakeAuthorizationService{iface: "xgs/6/2"}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+oltID.String()+"/authorize-onu",
		strings.NewReader(`{"port":"xgs/6","serial_number":"ISKT2308DD88"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !svc.called {
		t.Fatal("AuthorizeONU was not called")
	}
	if svc.gotOLTID != oltID || svc.gotPort != "xgs/6" || svc.gotSerial != "ISKT2308DD88" {
		t.Errorf("got oltID=%v port=%q serial=%q", svc.gotOLTID, svc.gotPort, svc.gotSerial)
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

func TestAuthorizeONUEndpointRejectsMissingPort(t *testing.T) {
	svc := &fakeAuthorizationService{}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
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

func TestAuthorizeONUEndpointRejectsMissingSerialNumber(t *testing.T) {
	svc := &fakeAuthorizationService{}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
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

func TestAuthorizeONUEndpointRejectsMalformedJSON(t *testing.T) {
	svc := &fakeAuthorizationService{}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
		strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthorizeONUEndpointRejectsInvalidOLTID(t *testing.T) {
	svc := &fakeAuthorizationService{}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/not-a-uuid/authorize-onu",
		strings.NewReader(`{"port":"xgs/6","serial_number":"ISKT2308DD88"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if svc.called {
		t.Error("service was called; it must never be reached for an invalid oltId")
	}
}

// TestPropagatesServiceErrorKinds proves each apperror.Kind
// AuthorizationService can return maps to the HTTP status
// httpx.WriteError already establishes for it — routing/wiring coverage,
// mirroring internal/diagnostics/kontron/httpapi's own
// TestPropagatesServiceErrorKinds.
func TestPropagatesServiceErrorKinds(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"unavailable", apperror.Unavailable("could not reach OLT", context.DeadlineExceeded), http.StatusServiceUnavailable},
		{"invalid", apperror.Invalid("serial number value contains a newline"), http.StatusBadRequest},
		{"conflict", apperror.Conflict("onu already authorized elsewhere"), http.StatusConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeAuthorizationService{err: tc.err}
			router := newTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
				strings.NewReader(`{"port":"xgs/6","serial_number":"ISKT2308DD88"}`))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}
