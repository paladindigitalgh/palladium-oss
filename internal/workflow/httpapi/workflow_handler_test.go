package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow/httpapi"
)

type stubWorkflowService struct {
	instance workflow.Instance
	err      error
}

func (s stubWorkflowService) Get(context.Context, uuid.UUID) (workflow.Instance, error) {
	return s.instance, s.err
}
func (s stubWorkflowService) List(context.Context) ([]workflow.Instance, error) { return nil, nil }
func (s stubWorkflowService) ListByServiceID(context.Context, uuid.UUID) ([]workflow.Instance, error) {
	return nil, nil
}
func (s stubWorkflowService) Create(_ context.Context, i workflow.Instance) (workflow.Instance, error) {
	return i, s.err
}
func (s stubWorkflowService) Delete(context.Context, uuid.UUID) error { return s.err }
func (s stubWorkflowService) Cancel(_ context.Context, id uuid.UUID) (workflow.Instance, error) {
	return workflow.Instance{ID: id, Status: workflow.StatusCancelled}, s.err
}
func (s stubWorkflowService) Retry(_ context.Context, id uuid.UUID) (workflow.Instance, error) {
	return workflow.Instance{ID: id, Status: workflow.StatusPending}, s.err
}

func withIDParam(req *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestCreateRejectsMalformedBody(t *testing.T) {
	h := httpapi.NewWorkflowHandler(stubWorkflowService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-instances", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateReturnsCreated(t *testing.T) {
	h := httpapi.NewWorkflowHandler(stubWorkflowService{})

	body := `{"service_id":"11111111-1111-1111-1111-111111111111","definition_name":"suspend-service"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-instances", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestGetRejectsInvalidID(t *testing.T) {
	h := httpapi.NewWorkflowHandler(stubWorkflowService{})

	req := withIDParam(httptest.NewRequest(http.MethodGet, "/api/v1/workflow-instances/not-a-uuid", nil), "not-a-uuid")
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
