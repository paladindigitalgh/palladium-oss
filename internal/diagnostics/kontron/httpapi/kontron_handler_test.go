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

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeKontronService is the seam httpapi.KontronHandler depends on (see
// its unexported kontronService interface in kontron_handler.go). It
// lets these tests exercise HTTP-only concerns — routing, status codes,
// JSON shapes, error translation — without a real service, Dialer, or
// SSH connection involved; internal/diagnostics/kontron/service and
// internal/diagnostics/kontron each have their own tests for the layers
// below this one.
type fakeKontronService struct {
	output       string
	err          error
	calledMethod string
	gotOLTID     uuid.UUID
	gotIface     string
}

func (f *fakeKontronService) result() (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.output, nil
}

func (f *fakeKontronService) ONUSummary(_ context.Context, oltID uuid.UUID) (string, error) {
	f.calledMethod, f.gotOLTID = "ONUSummary", oltID
	return f.result()
}
func (f *fakeKontronService) ONUStatusSummary(_ context.Context, oltID uuid.UUID) (string, error) {
	f.calledMethod, f.gotOLTID = "ONUStatusSummary", oltID
	return f.result()
}
func (f *fakeKontronService) ONURunningConfig(_ context.Context, oltID uuid.UUID, iface string) (string, error) {
	f.calledMethod, f.gotOLTID, f.gotIface = "ONURunningConfig", oltID, iface
	return f.result()
}
func (f *fakeKontronService) ONUDetail(_ context.Context, oltID uuid.UUID, iface string) (string, error) {
	f.calledMethod, f.gotOLTID, f.gotIface = "ONUDetail", oltID, iface
	return f.result()
}
func (f *fakeKontronService) ONUStatus(_ context.Context, oltID uuid.UUID, iface string) (string, error) {
	f.calledMethod, f.gotOLTID, f.gotIface = "ONUStatus", oltID, iface
	return f.result()
}
func (f *fakeKontronService) ONUEthernetPorts(_ context.Context, oltID uuid.UUID, iface string) (string, error) {
	f.calledMethod, f.gotOLTID, f.gotIface = "ONUEthernetPorts", oltID, iface
	return f.result()
}
func (f *fakeKontronService) DHCPSnoopingEntries(_ context.Context, oltID uuid.UUID, iface string) (string, error) {
	f.calledMethod, f.gotOLTID, f.gotIface = "DHCPSnoopingEntries", oltID, iface
	return f.result()
}
func (f *fakeKontronService) MACAddressTableEntries(_ context.Context, oltID uuid.UUID, iface string) (string, error) {
	f.calledMethod, f.gotOLTID, f.gotIface = "MACAddressTableEntries", oltID, iface
	return f.result()
}

// newTestRouter mounts a KontronHandler backed by svc on a real
// chi.Router, mirroring how internal/server/router.go mounts it in
// production (minus auth/authz — see authenticated_test.go for that).
func newTestRouter(svc *fakeKontronService) http.Handler {
	handler := httpapi.NewKontronHandler(svc)

	r := chi.NewRouter()
	r.Route("/diagnostics/olts/{oltId}", func(r chi.Router) {
		r.Post("/onu-summary", handler.ONUSummary)
		r.Post("/onu-status-summary", handler.ONUStatusSummary)
		r.Post("/onu-running-config", handler.ONURunningConfig)
		r.Post("/onu-detail", handler.ONUDetail)
		r.Post("/onu-status", handler.ONUStatus)
		r.Post("/onu-ethernet-ports", handler.ONUEthernetPorts)
		r.Post("/dhcp-snooping-entries", handler.DHCPSnoopingEntries)
		r.Post("/mac-address-table-entries", handler.MACAddressTableEntries)
	})
	return r
}

func decodeOutput(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Output string `json:"output"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body.Output
}

// TestNoArgEndpoints covers the two whole-OLT routes: no request body,
// oltId comes only from the path.
func TestNoArgEndpoints(t *testing.T) {
	cases := []struct {
		path       string
		wantMethod string
	}{
		{"/onu-summary", "ONUSummary"},
		{"/onu-status-summary", "ONUStatusSummary"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			oltID := uuid.New()
			svc := &fakeKontronService{output: "device output"}
			router := newTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/diagnostics/olts/"+oltID.String()+tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if decodeOutput(t, rec) != "device output" {
				t.Errorf("output = %q, want %q", decodeOutput(t, rec), "device output")
			}
			if svc.calledMethod != tc.wantMethod {
				t.Errorf("service method called = %q, want %q", svc.calledMethod, tc.wantMethod)
			}
			if svc.gotOLTID != oltID {
				t.Errorf("oltID passed to service = %v, want %v", svc.gotOLTID, oltID)
			}
		})
	}
}

// TestPerInterfaceEndpoints covers the six per-interface routes: a
// {"interface": "..."} body, oltId from the path.
func TestPerInterfaceEndpoints(t *testing.T) {
	cases := []struct {
		path       string
		wantMethod string
	}{
		{"/onu-running-config", "ONURunningConfig"},
		{"/onu-detail", "ONUDetail"},
		{"/onu-status", "ONUStatus"},
		{"/onu-ethernet-ports", "ONUEthernetPorts"},
		{"/dhcp-snooping-entries", "DHCPSnoopingEntries"},
		{"/mac-address-table-entries", "MACAddressTableEntries"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			oltID := uuid.New()
			svc := &fakeKontronService{output: "device output"}
			router := newTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/diagnostics/olts/"+oltID.String()+tc.path,
				strings.NewReader(`{"interface":"xgs/1/1"}`))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if decodeOutput(t, rec) != "device output" {
				t.Errorf("output = %q, want %q", decodeOutput(t, rec), "device output")
			}
			if svc.calledMethod != tc.wantMethod {
				t.Errorf("service method called = %q, want %q", svc.calledMethod, tc.wantMethod)
			}
			if svc.gotOLTID != oltID {
				t.Errorf("oltID passed to service = %v, want %v", svc.gotOLTID, oltID)
			}
			if svc.gotIface != "xgs/1/1" {
				t.Errorf("iface passed to service = %q, want %q", svc.gotIface, "xgs/1/1")
			}
		})
	}
}

func TestPerInterfaceEndpointRejectsEmptyInterface(t *testing.T) {
	svc := &fakeKontronService{output: "should never be reached"}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/olts/"+uuid.New().String()+"/onu-detail",
		strings.NewReader(`{"interface":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if svc.calledMethod != "" {
		t.Errorf("service was called (%s); it must never be reached for an empty interface", svc.calledMethod)
	}
}

func TestPerInterfaceEndpointRejectsMalformedJSON(t *testing.T) {
	svc := &fakeKontronService{output: "should never be reached"}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/olts/"+uuid.New().String()+"/onu-detail",
		strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestEndpointRejectsInvalidOLTID(t *testing.T) {
	svc := &fakeKontronService{output: "should never be reached"}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/olts/not-a-uuid/onu-summary", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if svc.calledMethod != "" {
		t.Errorf("service was called (%s); it must never be reached for an invalid oltId", svc.calledMethod)
	}
}

// TestPropagatesServiceErrorKinds proves each apperror.Kind
// KontronService can return (see that package's own classify function)
// maps to the HTTP status httpx.WriteError already establishes for it —
// this is routing/wiring coverage, not a re-test of WriteError's own
// mapping logic (see internal/httpx's own tests for that).
func TestPropagatesServiceErrorKinds(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", apperror.NotFound("olt not found"), http.StatusNotFound},
		{"conflict", apperror.Conflict("olt has no connection profile configured"), http.StatusConflict},
		{"unavailable", apperror.Unavailable("could not reach OLT", context.DeadlineExceeded), http.StatusServiceUnavailable},
		{"invalid", apperror.Invalid("interface value contains a newline"), http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeKontronService{err: tc.err}
			router := newTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/diagnostics/olts/"+uuid.New().String()+"/onu-summary", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}
