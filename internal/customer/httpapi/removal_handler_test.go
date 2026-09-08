package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	"github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/customer/removal"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

// fakeRemovalService is the seam httpapi.RemovalHandler depends on.
type fakeRemovalService struct {
	preview       removal.Preview
	previewErr    error
	executeErr    error
	gotPreviewID  uuid.UUID
	gotExecuteID  uuid.UUID
	executeCalled bool
}

func (f *fakeRemovalService) Preview(_ context.Context, customerID uuid.UUID) (removal.Preview, error) {
	f.gotPreviewID = customerID
	if f.previewErr != nil {
		return removal.Preview{}, f.previewErr
	}
	return f.preview, nil
}

func (f *fakeRemovalService) Execute(_ context.Context, customerID uuid.UUID) error {
	f.executeCalled = true
	f.gotExecuteID = customerID
	return f.executeErr
}

func newRemovalTestRouter(svc *fakeRemovalService) http.Handler {
	handler := httpapi.NewRemovalHandler(svc)

	r := chi.NewRouter()
	r.Route("/customers/{id}", func(r chi.Router) {
		r.Get("/removal-preview", handler.Preview)
		r.Post("/removal", handler.Execute)
	})
	return r
}

func newRemovalAuthenticatedTestRouter(svc *fakeRemovalService, tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	handler := httpapi.NewRemovalHandler(svc)
	authzMiddleware := authz.NewMiddleware(stubUserRepository{role: role})

	r := chi.NewRouter()
	r.Route("/customers/{id}", func(r chi.Router) {
		r.Use(auth.Middleware(tokens))
		r.Group(func(r chi.Router) {
			r.Use(authzMiddleware.RequireCustomerWrite())
			r.Get("/removal-preview", handler.Preview)
			r.Post("/removal", handler.Execute)
		})
	})
	return r
}

func TestRemovalPreviewEndpoint(t *testing.T) {
	customerID := uuid.New()
	svc := &fakeRemovalService{preview: removal.Preview{CustomerID: customerID}}
	router := newRemovalTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/customers/"+customerID.String()+"/removal-preview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if svc.gotPreviewID != customerID {
		t.Errorf("gotPreviewID = %v, want %v", svc.gotPreviewID, customerID)
	}
}

func TestRemovalPreviewEndpointRejectsInvalidID(t *testing.T) {
	svc := &fakeRemovalService{}
	router := newRemovalTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/customers/not-a-uuid/removal-preview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestRemovalExecuteEndpoint(t *testing.T) {
	customerID := uuid.New()
	svc := &fakeRemovalService{}
	router := newRemovalTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/customers/"+customerID.String()+"/removal", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if !svc.executeCalled || svc.gotExecuteID != customerID {
		t.Errorf("Execute called=%v gotExecuteID=%v, want called with %v", svc.executeCalled, svc.gotExecuteID, customerID)
	}
}

func TestRemovalEndpointsPropagateServiceErrorKinds(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"unavailable", apperror.Unavailable("could not reach OLT", context.DeadlineExceeded), http.StatusServiceUnavailable},
		{"not_found", apperror.NotFound("customer not found"), http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeRemovalService{executeErr: tc.err}
			router := newRemovalTestRouter(svc)

			req := httptest.NewRequest(http.MethodPost, "/customers/"+uuid.New().String()+"/removal", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

var removalAuthTestNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestViewerCannotExecuteRemoval(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(removalAuthTestNow))
	router := newRemovalAuthenticatedTestRouter(&fakeRemovalService{}, tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/customers/"+uuid.New().String()+"/removal", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestOperatorCanExecuteRemoval(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(removalAuthTestNow))
	router := newRemovalAuthenticatedTestRouter(&fakeRemovalService{}, tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/customers/"+uuid.New().String()+"/removal", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}
