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
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/httpapi"
)

// stubUserRepository satisfies auth.UserRepository structurally, always
// reporting the configured role for GetByID regardless of which ID is
// asked for — enough for authz.Middleware, which is all these tests need
// it for. Mirrors internal/serviceequipment/httpapi/authenticated_test.go's
// stub of the same name.
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

// newAuthenticatedTestRouter mounts ProvisioningHandler behind the real
// auth.Middleware and authz.Middleware, exactly as
// internal/server/router.go wires /api/v1/provisioning-jobs in
// production — the fake service and stub user repository are the only
// stand-ins; everything about how a request reaches the handler
// (routing, authentication, authorization, context propagation) is the
// genuine article. This is what distinguishes these tests from
// provisioning_handler_test.go's: those test the handler in isolation,
// with no middleware in front of it at all — and, specifically, with no
// real Claims in the request context, which is why
// TestCreateSetsRequestedByUserIDFromAuthenticatedCaller below can only
// be proven here.
func newAuthenticatedTestRouter(svc *fakeProvisioningService, tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	handler := httpapi.NewProvisioningHandler(svc)
	authzMiddleware := authz.NewMiddleware(stubUserRepository{role: role})

	r := chi.NewRouter()
	r.Route("/provisioning-jobs", func(r chi.Router) {
		r.Use(auth.Middleware(tokens))

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireProvisioningRead())
			r.Get("/", handler.List)
			r.Get("/{id}", handler.Get)
		})

		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireProvisioningWrite())
			r.Post("/", handler.Create)
			r.Delete("/{id}", handler.Delete)
			r.Post("/{id}/start", handler.Start)
			r.Post("/{id}/succeed", handler.Succeed)
			r.Post("/{id}/fail", handler.Fail)
			r.Post("/{id}/cancel", handler.Cancel)
			r.Post("/{id}/retry", handler.Retry)
		})
	})
	return r
}

var authTestNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func mustIssueTokenForUser(t *testing.T, tokens *auth.TokenIssuer, userID uuid.UUID) string {
	t.Helper()
	token, err := tokens.IssueToken(auth.User{ID: userID, Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}
	return token
}

func mustIssueToken(t *testing.T, tokens *auth.TokenIssuer) string {
	t.Helper()
	return mustIssueTokenForUser(t, tokens, uuid.New())
}

func TestUnauthenticatedRequestRejectedWithoutReachingHandler(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeProvisioningService(), tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/provisioning-jobs/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestViewerCanReadProvisioning is "apply the standard RBAC matrix",
// applied to Provisioning: Viewer can read.
func TestViewerCanReadProvisioning(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeProvisioningService(), tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestViewerCannotWriteProvisioning is "apply the standard RBAC matrix",
// applied to Provisioning: Viewer cannot write.
func TestViewerCannotWriteProvisioning(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeProvisioningService(), tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestOperatorCanWriteProvisioning is "apply the standard RBAC matrix",
// applied to Provisioning: Operator can write.
func TestOperatorCanWriteProvisioning(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeProvisioningService(), tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestAdministratorCanWriteProvisioning is "apply the standard RBAC
// matrix", applied to Provisioning: Administrator can write.
func TestAdministratorCanWriteProvisioning(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeProvisioningService(), tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestCreateSetsRequestedByUserIDFromAuthenticatedCaller proves
// ProvisioningHandler.Create takes RequestedByUserID from the real,
// JWT-authenticated caller's Claims — never from the request body (see
// provisioningJobCreateRequest's doc comment in dto.go) — by issuing a
// token for a specific UserID and checking the created job reports
// exactly that ID, with no such field present anywhere in the request.
func TestCreateSetsRequestedByUserIDFromAuthenticatedCaller(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(authTestNow))
	router := newAuthenticatedTestRouter(newFakeProvisioningService(), tokens, auth.RoleAdministrator)
	callerID := uuid.New()
	token := mustIssueTokenForUser(t, tokens, callerID)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		RequestedByUserID string `json:"requested_by_user_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RequestedByUserID != callerID.String() {
		t.Errorf("requested_by_user_id = %q, want %q (the authenticated caller's own ID)", body.RequestedByUserID, callerID.String())
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
	router := newAuthenticatedTestRouter(newFakeProvisioningService(), expiredValidator, auth.RoleAdministrator)

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
