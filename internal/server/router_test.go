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
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	locationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
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

// stubUserRepository satisfies auth.UserRepository structurally, always
// reporting the configured role for GetByID regardless of which ID is
// asked for — enough for authz.Middleware, which is all these router
// tests need it for.
type stubUserRepository struct {
	role auth.Role
}

func (s stubUserRepository) GetByID(context.Context, uuid.UUID) (auth.User, error) {
	return auth.User{Role: s.role}, nil
}
func (s stubUserRepository) GetByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented in this stub")
}
func (s stubUserRepository) Create(_ context.Context, u auth.User) (auth.User, error) { return u, nil }
func (s stubUserRepository) UpdatePasswordHash(context.Context, uuid.UUID, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented in this stub")
}
func (s stubUserRepository) Count(context.Context) (int, error) { return 0, nil }

var _ auth.UserRepository = stubUserRepository{}

// newRouterWithSites builds the real production router (api.NewRouter),
// with a stub service and a stub user repository standing in for the
// database, so these tests prove something no other test file can: that
// /api/v1/sites is actually wired up behind both auth.Middleware and
// authz.Middleware in this file's router.go, with the exact capability
// each HTTP method requires — not just that authentication and
// authorization work correctly in isolation (see
// internal/inventory/httpapi/authenticated_test.go and
// internal/authz/middleware_test.go for far more thorough versions of
// those checks). If someone editing router.go ever forgot to add
// RequireInventoryWrite() to the POST/PUT/DELETE group, or accidentally
// required it on GET too, these tests would catch it; the more isolated
// test files could not.
func newRouterWithSites(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:      logger,
		Version:     "test",
		Commit:      "test",
		SiteHandler: httpapi.NewSiteHandler(stubSiteService{}),
		Tokens:      tokens,
		Authz:       authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

func mustIssueToken(t *testing.T, tokens *auth.TokenIssuer) string {
	t.Helper()
	token, err := tokens.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}
	return token
}

func TestRouterRejectsUnauthenticatedSiteRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sites/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadSites is goal 7's "Viewer can read inventory",
// proven through the real, fully wired router.
func TestRouterViewerCanReadSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteSites is goal 7's "Viewer cannot modify
// inventory", proven through the real, fully wired router.
func TestRouterViewerCannotWriteSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/", strings.NewReader(`{"name":"Test Site"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteSites is goal 7's "Operator can modify
// inventory", proven through the real, fully wired router.
func TestRouterOperatorCanWriteSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/", strings.NewReader(`{"name":"Test Site"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteSites is goal 7's "Administrator can
// modify inventory", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/", strings.NewReader(`{"name":"Test Site"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubCustomerService satisfies whatever interface
// customerhttpapi.CustomerHandler needs structurally, the same technique
// stubSiteService above uses.
type stubCustomerService struct{}

func (stubCustomerService) Get(context.Context, uuid.UUID) (customer.Customer, error) {
	return customer.Customer{}, apperror.NotFound("customer not found")
}
func (stubCustomerService) List(context.Context) ([]customer.Customer, error) { return nil, nil }
func (stubCustomerService) Create(_ context.Context, c customer.Customer) (customer.Customer, error) {
	return c, nil
}
func (stubCustomerService) Update(_ context.Context, c customer.Customer) (customer.Customer, error) {
	return c, nil
}
func (stubCustomerService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithCustomers mirrors newRouterWithSites exactly, one resource
// over: it proves /api/v1/customers is wired up behind both
// auth.Middleware and authz.Middleware in the real production router,
// with RequireCustomerRead/RequireCustomerWrite applied to the right HTTP
// methods (goal 6: "apply the same authorization model as Sites"). See
// internal/customer/httpapi/authenticated_test.go for a far more thorough
// version of the same checks, scoped to that package.
func newRouterWithCustomers(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:          logger,
		Version:         "test",
		Commit:          "test",
		CustomerHandler: customerhttpapi.NewCustomerHandler(stubCustomerService{}),
		Tokens:          tokens,
		Authz:           authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validCustomerBody = `{"name":"Test Customer","customer_type":"Residential","status":"Active"}`

func TestRouterRejectsUnauthenticatedCustomerRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/customers/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadCustomers is goal 7's "Viewer can read
// inventory", applied to Customers, proven through the real, fully wired
// router.
func TestRouterViewerCanReadCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteCustomers is goal 7's "Viewer cannot modify
// inventory", applied to Customers, proven through the real, fully wired
// router.
func TestRouterViewerCannotWriteCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/", strings.NewReader(validCustomerBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteCustomers is goal 7's "Operator can modify
// inventory", applied to Customers, proven through the real, fully wired
// router.
func TestRouterOperatorCanWriteCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/", strings.NewReader(validCustomerBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteCustomers is goal 7's "Administrator can
// modify inventory", applied to Customers, proven through the real, fully
// wired router.
func TestRouterAdministratorCanWriteCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/", strings.NewReader(validCustomerBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubLocationService satisfies whatever interface
// locationhttpapi.LocationHandler needs structurally, the same technique
// stubSiteService and stubCustomerService above use.
type stubLocationService struct{}

func (stubLocationService) Get(context.Context, uuid.UUID) (location.Location, error) {
	return location.Location{}, apperror.NotFound("location not found")
}
func (stubLocationService) List(context.Context) ([]location.Location, error) { return nil, nil }
func (stubLocationService) Create(_ context.Context, l location.Location) (location.Location, error) {
	return l, nil
}
func (stubLocationService) Update(_ context.Context, l location.Location) (location.Location, error) {
	return l, nil
}
func (stubLocationService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithLocations mirrors newRouterWithCustomers exactly, one
// resource over: it proves /api/v1/locations is wired up behind both
// auth.Middleware and authz.Middleware in the real production router, with
// RequireLocationRead/RequireLocationWrite applied to the right HTTP
// methods ("match Customer permissions"). See
// internal/location/httpapi/authenticated_test.go for a far more thorough
// version of the same checks, scoped to that package.
func newRouterWithLocations(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:          logger,
		Version:         "test",
		Commit:          "test",
		LocationHandler: locationhttpapi.NewLocationHandler(stubLocationService{}),
		Tokens:          tokens,
		Authz:           authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validLocationBody = `{"customer_id":"11111111-1111-1111-1111-111111111111","name":"Test Location","type":"Service","status":"Active"}`

func TestRouterRejectsUnauthenticatedLocationRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/locations/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadLocations is "match Customer permissions",
// proven through the real, fully wired router.
func TestRouterViewerCanReadLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteLocations is "match Customer permissions",
// proven through the real, fully wired router.
func TestRouterViewerCannotWriteLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/locations/", strings.NewReader(validLocationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteLocations is "match Customer permissions",
// proven through the real, fully wired router.
func TestRouterOperatorCanWriteLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/locations/", strings.NewReader(validLocationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteLocations is "match Customer
// permissions", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/locations/", strings.NewReader(validLocationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
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
