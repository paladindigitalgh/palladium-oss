package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

func TestMiddlewareAllowsValidTokenAndStoresClaims(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))
	user := auth.User{ID: uuid.New(), Email: "jane@example.com"}
	token, err := tokens.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	var gotClaims auth.Claims
	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			t.Error("ClaimsFromContext() found no claims in the request seen by next")
		}
		gotClaims = claims
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	auth.Middleware(tokens)(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("next handler was not called for a valid token")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotClaims.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", gotClaims.UserID, user.ID)
	}
	if gotClaims.Email != user.Email {
		t.Errorf("Email = %q, want %q", gotClaims.Email, user.Email)
	}
}

func TestMiddlewareRejectsMissingAuthorizationHeader(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	assertMiddlewareRejects(t, tokens, req, rec)
}

func TestMiddlewareRejectsNonBearerScheme(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	assertMiddlewareRejects(t, tokens, req, rec)
}

func TestMiddlewareRejectsEmptyBearerToken(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	assertMiddlewareRejects(t, tokens, req, rec)
}

func TestMiddlewareRejectsExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	issuer := auth.NewTokenIssuer(secret, time.Hour, clock.NewFrozen(fixedNow))
	token, err := issuer.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	// Middleware validates using its own TokenIssuer's clock, so build one
	// frozen after the token's expiration (same technique as
	// token_test.go's TestParseTokenRejectsExpiredToken).
	afterExpiry := fixedNow.Add(2 * time.Hour)
	expiredValidator := auth.NewTokenIssuer(secret, time.Hour, clock.NewFrozen(afterExpiry))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	assertMiddlewareRejects(t, expiredValidator, req, rec)
}

func TestMiddlewareRejectsWrongSignature(t *testing.T) {
	issuer := auth.NewTokenIssuer([]byte("secret-a"), time.Hour, clock.NewFrozen(fixedNow))
	token, err := issuer.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	validator := auth.NewTokenIssuer([]byte("secret-b"), time.Hour, clock.NewFrozen(fixedNow))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	assertMiddlewareRejects(t, validator, req, rec)
}

func assertMiddlewareRejects(t *testing.T, tokens *auth.TokenIssuer, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	auth.Middleware(tokens)(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("next handler was called despite an invalid/missing token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
