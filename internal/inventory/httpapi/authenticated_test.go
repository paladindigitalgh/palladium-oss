package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

// newAuthenticatedTestRouter mounts SiteHandler behind the real
// auth.Middleware, exactly as internal/server/router.go wires
// /api/v1/sites in production — the fake service is the only stand-in;
// everything about how a request reaches it (routing, middleware,
// rejection, context propagation) is the genuine article. This is what
// distinguishes these tests from site_handler_test.go's: those test the
// handler in isolation, with no middleware in front of it at all.
func newAuthenticatedTestRouter(svc *fakeSiteService, tokens *auth.TokenIssuer) http.Handler {
	handler := httpapi.NewSiteHandler(svc)

	r := chi.NewRouter()
	r.Route("/sites", func(r chi.Router) {
		r.Use(auth.Middleware(tokens))
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Put("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	})
	return r
}

var authTestNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestAuthenticatedEndpointAllowsValidToken(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	token, err := tokens.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	router := newAuthenticatedTestRouter(newFakeSiteService(), tokens)

	req := httptest.NewRequest(http.MethodGet, "/sites/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUnauthorizedRequestRejectedWithoutReachingHandler(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	svc := newFakeSiteService()
	router := newAuthenticatedTestRouter(svc, tokens)

	cases := []struct {
		name   string
		mutate func(*http.Request)
	}{
		{"no header", func(*http.Request) {}},
		{"wrong scheme", func(r *http.Request) { r.Header.Set("Authorization", "Basic dXNlcjpwYXNz") }},
		{"malformed token", func(r *http.Request) { r.Header.Set("Authorization", "Bearer not-a-real-jwt") }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/sites/", nil)
			c.mutate(req)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
		})
	}
}

func TestUnauthorizedRequestRejectedForEveryMethod(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeSiteService(), tokens)

	requests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/sites/"},
		{http.MethodGet, "/sites/"},
		{http.MethodGet, "/sites/" + uuid.New().String()},
		{http.MethodPut, "/sites/" + uuid.New().String()},
		{http.MethodDelete, "/sites/" + uuid.New().String()},
	}

	for _, req := range requests {
		t.Run(req.method, func(t *testing.T) {
			r := httptest.NewRequest(req.method, req.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, r)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s status = %d, want %d", req.method, req.path, rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAuthenticatedEndpointRejectsExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	issuer := auth.NewTokenIssuer(secret, time.Hour, clock.NewFrozen(authTestNow))
	token, err := issuer.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	// The router is wired with a validator whose clock is frozen after
	// the token's expiration (same technique as
	// internal/auth/token_test.go's TestParseTokenRejectsExpiredToken).
	afterExpiry := authTestNow.Add(2 * time.Hour)
	expiredValidator := auth.NewTokenIssuer(secret, time.Hour, clock.NewFrozen(afterExpiry))
	router := newAuthenticatedTestRouter(newFakeSiteService(), expiredValidator)

	req := httptest.NewRequest(http.MethodGet, "/sites/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
