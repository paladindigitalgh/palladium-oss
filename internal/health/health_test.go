package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubChecker struct {
	name string
	err  error
}

func (s stubChecker) Name() string                    { return s.name }
func (s stubChecker) Check(ctx context.Context) error { return s.err }

func TestLiveAlwaysOK(t *testing.T) {
	h := NewHandler(nil, "1.2.3", "abcdef")

	rec := httptest.NewRecorder()
	h.Live(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["version"] != "1.2.3" {
		t.Errorf("version = %q, want %q", body["version"], "1.2.3")
	}
}

func TestReadyWithNoCheckersIsOK(t *testing.T) {
	h := NewHandler(nil, "dev", "none")

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyReportsFailingChecker(t *testing.T) {
	h := NewHandler([]Checker{
		stubChecker{name: "database", err: errors.New("connection refused")},
	}, "dev", "none")

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Checks["database"] != "connection refused" {
		t.Errorf("checks[database] = %q, want %q", body.Checks["database"], "connection refused")
	}
}
