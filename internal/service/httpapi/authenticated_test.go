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
	"github.com/paladindigitalgh/palladium-oss/internal/service/httpapi"
)

// stubUserRepository satisfies auth.UserRepository structurally, always
// reporting the configured role for GetByID regardless of which ID is
// asked for — enough for authz.Middleware, which is all these tests need
// it for. Mirrors internal/product/httpapi/authenticated_test.go's stub
// of the same name.
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

// newAuthenticatedTestRouter mounts ServiceHandler behind the real
// auth.Middleware and authz.Middleware, exactly as
// internal/server/router.go wires /api/v1/services in production — the
// fake service and stub user repository are the only stand-ins;
// everything about how a request reaches the handler (routing,
// authentication, authorization, context propagation) is the genuine
// article. This is what distinguishes these tests from
// service_handler_test.go's: those test the handler in isolation, with
// no middleware in front of it at all.
func newAuthenticatedTestRouter(svc *fakeServiceService, tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	handler := httpapi.NewServiceHandler(svc)
	authzMiddleware := authz.NewMiddleware(stubUserRepository{role: role})

	r := chi.NewRouter()
	r.Route("/services", func(r chi.Router) {
		r.Use(auth.Middleware(tokens))

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireServiceRead())
			r.Get("/", handler.List)
			r.Get("/{id}", handler.Get)
		})

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireServiceWrite())
			r.Post("/", handler.Create)
			r.Put("/{id}", handler.Update)
			r.Delete("/{id}", handler.Delete)
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

func TestUnauthenticatedRequestRejectedWithoutReachingHandler(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeServiceService(), tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/services/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestViewerCanReadServices is "apply the standard RBAC matrix", applied
// to Services: Viewer can read.
func TestViewerCanReadServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeServiceService(), tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/services/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestViewerCannotWriteServices is "apply the standard RBAC matrix",
// applied to Services: Viewer cannot write.
func TestViewerCannotWriteServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeServiceService(), tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/services/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestOperatorCanWriteServices is "apply the standard RBAC matrix",
// applied to Services: Operator can write.
func TestOperatorCanWriteServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeServiceService(), tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/services/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestAdministratorCanWriteServices is "apply the standard RBAC matrix",
// applied to Services: Administrator can write.
func TestAdministratorCanWriteServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeServiceService(), tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/services/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
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
	router := newAuthenticatedTestRouter(newFakeServiceService(), expiredValidator, auth.RoleAdministrator)

	req := httptest.NewRequest(http.MethodGet, "/services/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
