package api_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authhttpapi "github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
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

// stubSiteService satisfies whatever interface httpapi.SiteHandler needs
// structurally (Go interfaces need no explicit "implements" declaration),
// so these tests can build a real *httpapi.SiteHandler without a database.
type stubSiteService struct{}

func (stubSiteService) Get(context.Context, uuid.UUID) (inventory.Site, error) {
	return inventory.Site{}, apperror.NotFound("site not found")
}
func (stubSiteService) List(context.Context) ([]inventory.Site, error) { return nil, nil }
func (stubSiteService) Create(_ context.Context, s inventory.Site) (inventory.Site, error) {
	return s, nil
}
func (stubSiteService) Update(_ context.Context, s inventory.Site) (inventory.Site, error) {
	return s, nil
}
func (stubSiteService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithSites builds the real production router (api.NewRouter),
// with a stub service standing in for the database, so these tests prove
// something router_test.go's other test can't: that /api/v1/sites is
// actually wired up behind auth.Middleware in this file, not just that
// the middleware and handler work correctly in isolation (see
// internal/inventory/httpapi/authenticated_test.go for that — a much more
// thorough version of these same two checks, scoped to the httpapi
// package itself). If someone editing router.go ever forgot to add
// r.Use(auth.Middleware(...)) to the /sites group, that test file
// wouldn't catch it — this one would.
func newRouterWithSites(tokens *auth.TokenIssuer) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:      logger,
		Version:     "test",
		Commit:      "test",
		SiteHandler: httpapi.NewSiteHandler(stubSiteService{}),
		Tokens:      tokens,
	})
}

func TestRouterRejectsUnauthenticatedSiteRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sites/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouterAllowsAuthenticatedSiteRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	token, err := tokens.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}
	router := newRouterWithSites(tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// stubAuthService satisfies authhttpapi's unexported authService
// interface structurally, the same technique stubSiteService above uses.
type stubAuthService struct{}

func (stubAuthService) Authenticate(context.Context, string, string) (string, error) {
	return "stub.jwt.token", nil
}

// TestRouterLoginEndpointIsReachableWithoutAuthentication proves
// /api/v1/auth/login is wired up outside the auth.Middleware group in the
// real production router — a caller has no token yet at the point they
// are trying to obtain one, so this route must never require one. See
// internal/auth/httpapi's own tests for thorough coverage of the login
// handler's behavior in isolation; this test only proves the wiring.
func TestRouterLoginEndpointIsReachableWithoutAuthentication(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := api.NewRouter(api.Dependencies{
		Logger:       logger,
		Version:      "test",
		Commit:       "test",
		LoginHandler: authhttpapi.NewLoginHandler(stubAuthService{}, 30*time.Minute),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"whatever"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
