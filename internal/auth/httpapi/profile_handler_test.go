package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeProfileService is the seam httpapi.ProfileHandler depends on (see
// its unexported profileService interface in profile_handler.go). It
// lets these tests exercise HTTP-only concerns -- status codes, JSON
// shapes, claims-to-caller-ID resolution -- without a real service,
// repository, or database; internal/auth/service has its own tests for
// the business logic below this layer.
type fakeProfileService struct {
	user auth.User
	err  error

	gotUpdateNameID      uuid.UUID
	gotUpdateNameFirst   string
	gotUpdateNameLast    string
	gotChangePasswordID  uuid.UUID
	gotChangePasswordOld string
	gotChangePasswordNew string
}

func (f *fakeProfileService) Get(_ context.Context, id uuid.UUID) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	return f.user, nil
}

func (f *fakeProfileService) UpdateName(_ context.Context, id uuid.UUID, firstName, lastName string) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	f.gotUpdateNameID = id
	f.gotUpdateNameFirst = firstName
	f.gotUpdateNameLast = lastName
	f.user.FirstName = firstName
	f.user.LastName = lastName
	return f.user, nil
}

func (f *fakeProfileService) ChangePassword(_ context.Context, id uuid.UUID, currentPassword, newPassword string) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	f.gotChangePasswordID = id
	f.gotChangePasswordOld = currentPassword
	f.gotChangePasswordNew = newPassword
	return f.user, nil
}

func requestWithClaims(method, path, body string, claims auth.Claims) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	return r.WithContext(auth.ContextWithClaims(r.Context(), claims))
}

func TestProfileHandlerGetRequiresClaims(t *testing.T) {
	h := httpapi.NewProfileHandler(&fakeProfileService{})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestProfileHandlerGetReturnsCaller(t *testing.T) {
	claims := auth.Claims{UserID: uuid.New(), Email: "jane@example.com"}
	svc := &fakeProfileService{user: auth.User{ID: claims.UserID, Email: claims.Email, FirstName: "Jane", LastName: "Doe"}}
	h := httpapi.NewProfileHandler(svc)

	req := requestWithClaims(http.MethodGet, "/me", "", claims)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Email != "jane@example.com" || resp.FirstName != "Jane" || resp.LastName != "Doe" {
		t.Errorf("response = %+v, want jane@example.com/Jane/Doe", resp)
	}
}

func TestProfileHandlerUpdateNameActsOnCaller(t *testing.T) {
	claims := auth.Claims{UserID: uuid.New(), Email: "jane@example.com"}
	svc := &fakeProfileService{user: auth.User{ID: claims.UserID, Email: claims.Email}}
	h := httpapi.NewProfileHandler(svc)

	req := requestWithClaims(http.MethodPut, "/me", `{"first_name":"Jane","last_name":"Doe"}`, claims)
	rec := httptest.NewRecorder()
	h.UpdateName(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if svc.gotUpdateNameID != claims.UserID {
		t.Errorf("UpdateName() acted on %v, want caller's own ID %v", svc.gotUpdateNameID, claims.UserID)
	}
	if svc.gotUpdateNameFirst != "Jane" || svc.gotUpdateNameLast != "Doe" {
		t.Errorf("name = %q %q, want Jane Doe", svc.gotUpdateNameFirst, svc.gotUpdateNameLast)
	}
}

func TestProfileHandlerChangePasswordActsOnCaller(t *testing.T) {
	claims := auth.Claims{UserID: uuid.New(), Email: "jane@example.com"}
	svc := &fakeProfileService{user: auth.User{ID: claims.UserID, Email: claims.Email}}
	h := httpapi.NewProfileHandler(svc)

	req := requestWithClaims(http.MethodPut, "/me/password", `{"current_password":"old","new_password":"new"}`, claims)
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if svc.gotChangePasswordID != claims.UserID {
		t.Errorf("ChangePassword() acted on %v, want caller's own ID %v", svc.gotChangePasswordID, claims.UserID)
	}
	if svc.gotChangePasswordOld != "old" || svc.gotChangePasswordNew != "new" {
		t.Errorf("passwords = %q/%q, want old/new", svc.gotChangePasswordOld, svc.gotChangePasswordNew)
	}
}

func TestProfileHandlerChangePasswordPropagatesServiceError(t *testing.T) {
	claims := auth.Claims{UserID: uuid.New(), Email: "jane@example.com"}
	svc := &fakeProfileService{err: apperror.Invalid("current password is incorrect")}
	h := httpapi.NewProfileHandler(svc)

	req := requestWithClaims(http.MethodPut, "/me/password", `{"current_password":"wrong","new_password":"new"}`, claims)
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
