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
	"github.com/paladindigitalgh/palladium-oss/internal/contact/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

// stubUserRepository satisfies auth.UserRepository structurally, always
// reporting the configured role for GetByID regardless of which ID is
// asked for — enough for authz.Middleware, which is all these tests need
// it for. Mirrors internal/location/httpapi/authenticated_test.go's stub
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

// newAuthenticatedTestRouter mounts ContactHandler behind the real
// auth.Middleware and authz.Middleware, exactly as
// internal/server/router.go wires /api/v1/contacts in production — the
// fake service and stub user repository are the only stand-ins;
// everything about how a request reaches the handler (routing,
// authentication, authorization, context propagation) is the genuine
// article. This is what distinguishes these tests from
// contact_handler_test.go's: those test the handler in isolation, with
// no middleware in front of it at all.
func newAuthenticatedTestRouter(svc *fakeContactService, tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	handler := httpapi.NewContactHandler(svc)
	authzMiddleware := authz.NewMiddleware(stubUserRepository{role: role})

	r := chi.NewRouter()
	r.Route("/contacts", func(r chi.Router) {
		r.Use(auth.Middleware(tokens))

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireContactRead())
			r.Get("/", handler.List)
			r.Get("/{id}", handler.Get)
		})

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireContactWrite())
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
	router := newAuthenticatedTestRouter(newFakeContactService(), tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/contacts/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestViewerCanReadContacts is "match Customer/Location permissions",
// applied to Contacts: Viewer can read.
func TestViewerCanReadContacts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeContactService(), tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/contacts/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestViewerCannotWriteContacts is "match Customer/Location
// permissions", applied to Contacts: Viewer cannot write.
func TestViewerCannotWriteContacts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeContactService(), tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/contacts/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestOperatorCanWriteContacts is "match Customer/Location permissions",
// applied to Contacts: Operator can write.
func TestOperatorCanWriteContacts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeContactService(), tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/contacts/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestAdministratorCanWriteContacts is "match Customer/Location
// permissions", applied to Contacts: Administrator can write.
func TestAdministratorCanWriteContacts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeContactService(), tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/contacts/", strings.NewReader(validBody))
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
	router := newAuthenticatedTestRouter(newFakeContactService(), expiredValidator, auth.RoleAdministrator)

	req := httptest.NewRequest(http.MethodGet, "/contacts/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
