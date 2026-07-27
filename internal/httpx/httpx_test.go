package httpx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

func TestWriteJSONSetsContentTypeAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()

	httpx.WriteJSON(rec, http.StatusCreated, map[string]string{"hello": "world"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["hello"] != "world" {
		t.Errorf("body = %v, want {hello: world}", body)
	}
}

func TestDecodeJSONRejectsMalformedBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{not valid json`))

	var dst map[string]string
	err := httpx.DecodeJSON(req, &dst)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a","surprise":"field"}`))

	var dst struct {
		Name string `json:"name"`
	}
	err := httpx.DecodeJSON(req, &dst)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}

func TestDecodeJSONAcceptsValidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Main Office"}`))

	var dst struct {
		Name string `json:"name"`
	}
	if err := httpx.DecodeJSON(req, &dst); err != nil {
		t.Fatalf("DecodeJSON() = %v, want nil", err)
	}
	if dst.Name != "Main Office" {
		t.Errorf("Name = %q, want %q", dst.Name, "Main Office")
	}
}

func TestWriteErrorMapsKindsToStatusCodes(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
	}{
		{apperror.Invalid("bad input"), http.StatusBadRequest},
		{apperror.NotFound("site abc not found"), http.StatusNotFound},
		{apperror.Conflict("already exists"), http.StatusConflict},
		{apperror.Unauthorized("invalid token"), http.StatusUnauthorized},
		{apperror.Forbidden("not permitted"), http.StatusForbidden},
		{apperror.Unavailable("db down", errors.New("dial tcp: refused")), http.StatusServiceUnavailable},
		{apperror.Internal("create site", errors.New("boom")), http.StatusInternalServerError},
		{errors.New("some unclassified error"), http.StatusInternalServerError},
	}

	for _, c := range cases {
		rec := httptest.NewRecorder()
		httpx.WriteError(rec, c.err)
		if rec.Code != c.wantStatus {
			t.Errorf("WriteError(%v) status = %d, want %d", c.err, rec.Code, c.wantStatus)
		}
	}
}

func TestWriteErrorBodyIncludesKindAndMessage(t *testing.T) {
	rec := httptest.NewRecorder()

	httpx.WriteError(rec, apperror.NotFound("site abc-123 not found"))

	var body struct {
		Error string `json:"error"`
		Kind  string `json:"kind"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Kind != string(apperror.KindNotFound) {
		t.Errorf("kind = %q, want %q", body.Kind, apperror.KindNotFound)
	}
	if body.Error != "site abc-123 not found" {
		t.Errorf("error = %q, want %q", body.Error, "site abc-123 not found")
	}
}

// TestWriteErrorNeverLeaksWrappedCause is the concrete check behind goal
// 5's "keep PostgreSQL errors hidden from clients": an apperror.Internal
// (exactly what internal/inventory/postgres/errors.go's translateError
// produces for an unrecognized database error) wraps its cause, and that
// cause's text must never reach the response body — even though the cause
// contains something that looks exactly like what a real driver error
// would say.
func TestWriteErrorNeverLeaksWrappedCause(t *testing.T) {
	sensitive := errors.New(`pq: password authentication failed for user "palladium" at host 10.0.4.12`)
	err := apperror.Internal("create site", sensitive)

	rec := httptest.NewRecorder()
	httpx.WriteError(rec, err)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	body := rec.Body.String()
	if strings.Contains(body, "palladium") || strings.Contains(body, "10.0.4.12") || strings.Contains(body, "pq:") {
		t.Fatalf("response body leaked the wrapped cause: %s", body)
	}

	var decoded struct {
		Error string `json:"error"`
		Kind  string `json:"kind"`
	}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.Error != "an internal error occurred" {
		t.Errorf("error = %q, want the generic internal message", decoded.Error)
	}
}

// TestWriteErrorNeverLeaksWrappedCauseForUnavailable mirrors the test
// above for KindUnavailable, the other kind that wraps a cause (see
// internal/database.Pool.WarmUp's use of retry, and any future dependency
// health check).
func TestWriteErrorNeverLeaksWrappedCauseForUnavailable(t *testing.T) {
	sensitive := errors.New("dial tcp 10.0.4.12:5432: connect: connection refused")
	err := apperror.Unavailable("database ping failed", sensitive)

	rec := httptest.NewRecorder()
	httpx.WriteError(rec, err)

	if strings.Contains(rec.Body.String(), "10.0.4.12") {
		t.Fatalf("response body leaked the wrapped cause: %s", rec.Body.String())
	}
}
