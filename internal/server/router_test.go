package api_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/paladindigitalgh/palladium-oss/internal/server"
)

func TestRouterMountsInventorySchemaEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := api.NewRouter(api.Dependencies{
		Logger:  logger,
		Version: "test",
		Commit:  "test",
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/inventory/schema", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}
