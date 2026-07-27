package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAuthService is the seam httpapi.LoginHandler depends on (see its
// unexported authService interface in login_handler.go). It lets these
// tests exercise HTTP-only concerns — status codes, JSON shapes, error
// translation — without a real AuthService, UserRepository, or database;
// internal/auth/service_test.go already covers AuthService.Authenticate's
// own behavior.
type fakeAuthService struct {
	token string
	err   error
}

func (f *fakeAuthService) Authenticate(context.Context, string, string) (string, error) {
	return f.token, f.err
}

func TestLoginHandlerSuccess(t *testing.T) {
	svc := &fakeAuthService{token: "signed.jwt.token"}
	handler := httpapi.NewLoginHandler(svc, 30*time.Minute)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery staple"}`))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
		ExpiresIn   int    `json:"expiresIn"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AccessToken != "signed.jwt.token" {
		t.Errorf("accessToken = %q, want %q", body.AccessToken, "signed.jwt.token")
	}
	if body.TokenType != "Bearer" {
		t.Errorf("tokenType = %q, want %q", body.TokenType, "Bearer")
	}
	if body.ExpiresIn != 1800 {
		t.Errorf("expiresIn = %d, want 1800 (30 minutes in seconds)", body.ExpiresIn)
	}
}

func TestLoginHandlerResponseNeverIncludesAPasswordField(t *testing.T) {
	svc := &fakeAuthService{token: "signed.jwt.token"}
	handler := httpapi.NewLoginHandler(svc, 30*time.Minute)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"correct horse battery staple"}`))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	body := rec.Body.String()
	if strings.Contains(strings.ToLower(body), "password") || strings.Contains(strings.ToLower(body), "hash") {
		t.Fatalf("response leaked a password/hash field: %s", body)
	}
}

func TestLoginHandlerUnknownUser(t *testing.T) {
	svc := &fakeAuthService{err: apperror.Unauthorized("invalid email or password")}
	handler := httpapi.NewLoginHandler(svc, 30*time.Minute)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"nobody@example.com","password":"whatever"}`))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestLoginHandlerInvalidPassword(t *testing.T) {
	svc := &fakeAuthService{err: apperror.Unauthorized("invalid email or password")}
	handler := httpapi.NewLoginHandler(svc, 30*time.Minute)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"wrong"}`))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestLoginHandlerUnknownUserAndInvalidPasswordAreIndistinguishable is the
// HTTP-layer version of internal/auth/service_test.go's
// TestAuthServiceAuthenticateDoesNotRevealWhichCaseOccurred: since
// AuthService already collapses both cases into one identical error (see
// its doc comment), and LoginHandler does no case-specific handling of
// its own, the two responses this handler produces must be byte-for-byte
// identical too — proving the anti-enumeration guarantee actually reaches
// the wire, not just the Go error value.
func TestLoginHandlerUnknownUserAndInvalidPasswordAreIndistinguishable(t *testing.T) {
	svc := &fakeAuthService{err: apperror.Unauthorized("invalid email or password")}
	handler := httpapi.NewLoginHandler(svc, 30*time.Minute)

	unknownUserReq := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"nobody@example.com","password":"whatever"}`))
	unknownUserRec := httptest.NewRecorder()
	handler.Login(unknownUserRec, unknownUserReq)

	wrongPasswordReq := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"wrong"}`))
	wrongPasswordRec := httptest.NewRecorder()
	handler.Login(wrongPasswordRec, wrongPasswordReq)

	if unknownUserRec.Code != wrongPasswordRec.Code {
		t.Errorf("status codes differ: unknown user = %d, wrong password = %d",
			unknownUserRec.Code, wrongPasswordRec.Code)
	}
	if unknownUserRec.Body.String() != wrongPasswordRec.Body.String() {
		t.Errorf("response bodies differ: unknown user = %q, wrong password = %q",
			unknownUserRec.Body.String(), wrongPasswordRec.Body.String())
	}
}

func TestLoginHandlerMalformedRequest(t *testing.T) {
	svc := &fakeAuthService{}
	handler := httpapi.NewLoginHandler(svc, 30*time.Minute)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{not valid json`))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestLoginHandlerRejectsUnknownFields(t *testing.T) {
	svc := &fakeAuthService{token: "signed.jwt.token"}
	handler := httpapi.NewLoginHandler(svc, 30*time.Minute)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"x","role":"admin"}`))
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
