package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/httpapi"
)

// stubUserRepository mirrors
// internal/diagnostics/kontron/httpapi/authenticated_test.go's own stub
// of the same name exactly — satisfies auth.UserRepository
// structurally, always reporting the configured role for GetByID
// regardless of which ID is asked for.
type stubUserRepository struct {
	role auth.Role
}

func (s stubUserRepository) GetByID(context.Context, uuid.UUID) (auth.User, error) {
	return auth.User{Role: s.role, Status: auth.UserStatusActive}, nil
}
func (s stubUserRepository) GetByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented in this stub")
}
func (s stubUserRepository) Create(_ context.Context, u auth.User) (auth.User, error) { return u, nil }
func (s stubUserRepository) UpdatePasswordHash(context.Context, uuid.UUID, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented in this stub")
}
func (s stubUserRepository) List(context.Context) ([]auth.User, error) { return nil, nil }
func (s stubUserRepository) UpdateRole(_ context.Context, _ uuid.UUID, role auth.Role) (auth.User, error) {
	return auth.User{Role: role, Status: auth.UserStatusActive}, nil
}
func (s stubUserRepository) UpdateStatus(_ context.Context, _ uuid.UUID, status auth.UserStatus) (auth.User, error) {
	return auth.User{Role: s.role, Status: status}, nil
}
func (s stubUserRepository) Count(context.Context) (int, error) { return 0, nil }

var _ auth.UserRepository = stubUserRepository{}

// newAuthenticatedTestRouter mounts AuthorizationHandler behind the real
// auth.Middleware and authz.Middleware, exactly as
// internal/server/router.go wires /api/v1/provisioning/olts/{oltId}/...
// in production, using RequireProvisioning — the capability
// distinguishing this route from every RequireDiagnostics-guarded one
// (see authz.CanRunProvisioning's own doc comment for why).
func newAuthenticatedTestRouter(svc *fakeAuthorizationService, tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	handler := httpapi.NewAuthorizationHandler(svc)
	authzMiddleware := authz.NewMiddleware(stubUserRepository{role: role})

	r := chi.NewRouter()
	r.Route("/provisioning/olts/{oltId}", func(r chi.Router) {
		r.Use(auth.Middleware(tokens))

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireProvisioning())
			r.Post("/authorize-onu", handler.AuthorizeONU)
		})
	})
	return r
}

var authTestNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func mustIssueToken(t *testing.T, tokens *auth.TokenIssuer) string {
	t.Helper()
	token, err := tokens.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}
	return token
}

func authorizeONURequestBody() string {
	return `{"port":"xgs/6","serial_number":"ISKT2308DD88"}`
}

// newDeauthorizationAuthenticatedTestRouter mirrors
// newAuthenticatedTestRouter for DeauthorizationHandler, exactly as
// internal/server/router.go wires
// /api/v1/provisioning/devices/{deviceId}/... in production.
func newDeauthorizationAuthenticatedTestRouter(svc *fakeDeauthorizationService, tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	handler := httpapi.NewDeauthorizationHandler(svc)
	authzMiddleware := authz.NewMiddleware(stubUserRepository{role: role})

	r := chi.NewRouter()
	r.Route("/provisioning/devices/{deviceId}", func(r chi.Router) {
		r.Use(auth.Middleware(tokens))

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireProvisioning())
			r.Post("/deauthorize-onu", handler.DeauthorizeONU)
		})
	})
	return r
}

func TestUnauthenticatedRequestRejectedWithoutReachingHandler(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(&fakeAuthorizationService{iface: "xgs/6/2"}, tokens, auth.RoleAdministrator)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
		nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestViewerCannotAuthorizeONU is "apply the standard RBAC matrix",
// applied here: Viewer cannot run a live provisioning action.
func TestViewerCannotAuthorizeONU(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(&fakeAuthorizationService{iface: "xgs/6/2"}, tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
		strings.NewReader(authorizeONURequestBody()))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestOperatorCanAuthorizeONU is "apply the standard RBAC matrix",
// applied here: Operator can run.
func TestOperatorCanAuthorizeONU(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(&fakeAuthorizationService{iface: "xgs/6/2"}, tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
		strings.NewReader(authorizeONURequestBody()))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestAdministratorCanAuthorizeONU is "apply the standard RBAC matrix",
// applied here: Administrator can run.
func TestAdministratorCanAuthorizeONU(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(&fakeAuthorizationService{iface: "xgs/6/2"}, tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/olts/"+uuid.New().String()+"/authorize-onu",
		strings.NewReader(authorizeONURequestBody()))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestViewerCannotDeauthorizeONU is "apply the standard RBAC matrix",
// applied here: Viewer cannot run a live provisioning action.
func TestViewerCannotDeauthorizeONU(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newDeauthorizationAuthenticatedTestRouter(&fakeDeauthorizationService{iface: "xgs/6/2"}, tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/devices/"+uuid.New().String()+"/deauthorize-onu", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestOperatorCanDeauthorizeONU is "apply the standard RBAC matrix",
// applied here: Operator can run.
func TestOperatorCanDeauthorizeONU(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newDeauthorizationAuthenticatedTestRouter(&fakeDeauthorizationService{iface: "xgs/6/2"}, tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning/devices/"+uuid.New().String()+"/deauthorize-onu", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
