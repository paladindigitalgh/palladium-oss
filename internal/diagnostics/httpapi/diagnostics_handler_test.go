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

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeDiagnosticsService is the seam httpapi.DiagnosticsHandler depends
// on (see its unexported diagnosticsService interface in
// diagnostics_handler.go). It lets these tests exercise HTTP-only
// concerns — status codes, JSON shapes, error translation — without a
// real service or Registry involved; internal/diagnostics/service and
// internal/diagnostics each have their own tests for the layers below
// this one.
type fakeDiagnosticsService struct {
	result   *diagnostics.Result
	err      error
	lastName string
	lastReq  diagnostics.Request
}

func (f *fakeDiagnosticsService) Run(_ context.Context, name string, request diagnostics.Request) (*diagnostics.Result, error) {
	f.lastName = name
	f.lastReq = request
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// newTestRouter mounts a DiagnosticsHandler backed by svc on a real
// chi.Router, the same pattern every other domain's handler test in
// this codebase uses.
func newTestRouter(svc *fakeDiagnosticsService) http.Handler {
	handler := httpapi.NewDiagnosticsHandler(svc)

	r := chi.NewRouter()
	r.Post("/diagnostics/basic-onu-check", handler.BasicONUCheck)
	return r
}

const validBody = `{"onuId":"11111111-1111-1111-1111-111111111111"}`

func placeholderResult() *diagnostics.Result {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &diagnostics.Result{
		Name:       diagnostics.BasicONUCheckName,
		StartedAt:  now,
		FinishedAt: now,
		Duration:   0,
		Sections: []diagnostics.Section{
			{Name: diagnostics.BasicONUCheckName, Command: "not implemented", Output: "not implemented"},
		},
	}
}

func TestDiagnosticsHandlerBasicONUCheck(t *testing.T) {
	svc := &fakeDiagnosticsService{result: placeholderResult()}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/basic-onu-check", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name     string `json:"name"`
		Duration string `json:"duration"`
		Sections []struct {
			Name    string `json:"name"`
			Command string `json:"command"`
			Output  string `json:"output"`
		} `json:"sections"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != diagnostics.BasicONUCheckName {
		t.Errorf("name = %q, want %q", body.Name, diagnostics.BasicONUCheckName)
	}
	if len(body.Sections) != 1 {
		t.Fatalf("len(sections) = %d, want 1", len(body.Sections))
	}
	if body.Sections[0].Command != "not implemented" {
		t.Errorf("sections[0].command = %q, want %q", body.Sections[0].Command, "not implemented")
	}
	if body.Sections[0].Output != "not implemented" {
		t.Errorf("sections[0].output = %q, want %q", body.Sections[0].Output, "not implemented")
	}
}

// TestDiagnosticsHandlerBasicONUCheckPassesONUIDThrough proves the
// handler decodes the camelCase "onuId" field (goal 7's exact input
// shape) and passes it through to the service, and that it always asks
// for diagnostics.BasicONUCheckName specifically (see
// DiagnosticsHandler's doc comment on why the name is not
// caller-supplied).
func TestDiagnosticsHandlerBasicONUCheckPassesONUIDThrough(t *testing.T) {
	svc := &fakeDiagnosticsService{result: placeholderResult()}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/basic-onu-check", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if svc.lastName != diagnostics.BasicONUCheckName {
		t.Errorf("service was asked to run %q, want %q", svc.lastName, diagnostics.BasicONUCheckName)
	}
	if svc.lastReq.ONUID.String() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("ONUID passed to service = %v, want %v", svc.lastReq.ONUID, "11111111-1111-1111-1111-111111111111")
	}
}

func TestDiagnosticsHandlerBasicONUCheckRejectsMalformedJSON(t *testing.T) {
	svc := &fakeDiagnosticsService{result: placeholderResult()}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/basic-onu-check", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDiagnosticsHandlerBasicONUCheckRejectsUnknownFields(t *testing.T) {
	svc := &fakeDiagnosticsService{result: placeholderResult()}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/basic-onu-check",
		strings.NewReader(`{"onu_id":"11111111-1111-1111-1111-111111111111"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; snake_case onu_id must be rejected — the field is onuId", rec.Code, http.StatusBadRequest)
	}
}

func TestDiagnosticsHandlerBasicONUCheckPropagatesServiceError(t *testing.T) {
	svc := &fakeDiagnosticsService{err: apperror.NotFound(`diagnostic "Basic ONU Check" not found`)}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/diagnostics/basic-onu-check", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
