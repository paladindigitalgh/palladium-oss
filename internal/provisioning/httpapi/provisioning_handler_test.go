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

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/httpapi"
)

// fakeProvisioningService is the seam httpapi.ProvisioningHandler
// depends on (see its unexported provisioningService interface in
// provisioning_handler.go). It lets these tests exercise HTTP-only
// concerns — status codes, JSON shapes, routing, error translation —
// without a real service, repository, or database;
// internal/provisioning/service and internal/provisioning/postgres each
// have their own tests for the layers below this one, including
// internal/provisioning/service's own tests for the state machine
// itself.
type fakeProvisioningService struct {
	jobs map[uuid.UUID]provisioning.ProvisioningJob
	err  error // if set, every method returns this error instead
}

func newFakeProvisioningService(jobs ...provisioning.ProvisioningJob) *fakeProvisioningService {
	f := &fakeProvisioningService{jobs: make(map[uuid.UUID]provisioning.ProvisioningJob)}
	for _, j := range jobs {
		f.jobs[j.ID] = j
	}
	return f
}

func (f *fakeProvisioningService) Get(_ context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	if f.err != nil {
		return provisioning.ProvisioningJob{}, f.err
	}
	j, ok := f.jobs[id]
	if !ok {
		return provisioning.ProvisioningJob{}, apperror.NotFound("provisioning job not found")
	}
	return j, nil
}

func (f *fakeProvisioningService) List(context.Context) ([]provisioning.ProvisioningJob, error) {
	if f.err != nil {
		return nil, f.err
	}
	jobs := make([]provisioning.ProvisioningJob, 0, len(f.jobs))
	for _, j := range f.jobs {
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (f *fakeProvisioningService) ListByServiceID(_ context.Context, serviceID uuid.UUID) ([]provisioning.ProvisioningJob, error) {
	if f.err != nil {
		return nil, f.err
	}
	var jobs []provisioning.ProvisioningJob
	for _, j := range f.jobs {
		if j.ServiceID == serviceID {
			jobs = append(jobs, j)
		}
	}
	return jobs, nil
}

func (f *fakeProvisioningService) Create(_ context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	if f.err != nil {
		return provisioning.ProvisioningJob{}, f.err
	}
	j.ID = uuid.New()
	j.Status = provisioning.ProvisioningStatusPending
	j.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	j.UpdatedAt = j.CreatedAt
	f.jobs[j.ID] = j
	return j, nil
}

func (f *fakeProvisioningService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.jobs[id]; !ok {
		return apperror.NotFound("provisioning job not found")
	}
	delete(f.jobs, id)
	return nil
}

func (f *fakeProvisioningService) transition(id uuid.UUID, status provisioning.ProvisioningStatus) (provisioning.ProvisioningJob, error) {
	if f.err != nil {
		return provisioning.ProvisioningJob{}, f.err
	}
	j, ok := f.jobs[id]
	if !ok {
		return provisioning.ProvisioningJob{}, apperror.NotFound("provisioning job not found")
	}
	j.Status = status
	f.jobs[id] = j
	return j, nil
}

func (f *fakeProvisioningService) Start(_ context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return f.transition(id, provisioning.ProvisioningStatusRunning)
}

func (f *fakeProvisioningService) Succeed(_ context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return f.transition(id, provisioning.ProvisioningStatusSucceeded)
}

func (f *fakeProvisioningService) Fail(_ context.Context, id uuid.UUID, errorMessage string) (provisioning.ProvisioningJob, error) {
	if f.err != nil {
		return provisioning.ProvisioningJob{}, f.err
	}
	j, ok := f.jobs[id]
	if !ok {
		return provisioning.ProvisioningJob{}, apperror.NotFound("provisioning job not found")
	}
	j.Status = provisioning.ProvisioningStatusFailed
	j.ErrorMessage = &errorMessage
	f.jobs[id] = j
	return j, nil
}

func (f *fakeProvisioningService) Cancel(_ context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	return f.transition(id, provisioning.ProvisioningStatusCancelled)
}

func (f *fakeProvisioningService) Retry(_ context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	if f.err != nil {
		return provisioning.ProvisioningJob{}, f.err
	}
	j, ok := f.jobs[id]
	if !ok {
		return provisioning.ProvisioningJob{}, apperror.NotFound("provisioning job not found")
	}
	j.Status = provisioning.ProvisioningStatusPending
	j.RetryCount++
	f.jobs[id] = j
	return j, nil
}

// newTestRouter mounts a ProvisioningHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter get one populated
// the same way production code does, rather than faking chi's route
// context by hand.
func newTestRouter(svc *fakeProvisioningService) http.Handler {
	handler := httpapi.NewProvisioningHandler(svc)

	r := chi.NewRouter()
	r.Post("/provisioning-jobs", handler.Create)
	r.Get("/provisioning-jobs", handler.List)
	r.Get("/provisioning-jobs/{id}", handler.Get)
	r.Delete("/provisioning-jobs/{id}", handler.Delete)
	r.Post("/provisioning-jobs/{id}/start", handler.Start)
	r.Post("/provisioning-jobs/{id}/succeed", handler.Succeed)
	r.Post("/provisioning-jobs/{id}/fail", handler.Fail)
	r.Post("/provisioning-jobs/{id}/cancel", handler.Cancel)
	r.Post("/provisioning-jobs/{id}/retry", handler.Retry)
	return r
}

const validBody = `{"service_id":"11111111-1111-1111-1111-111111111111","operation":"Provision"}`

func TestProvisioningHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeProvisioningService())

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID                string  `json:"id"`
		ServiceID         string  `json:"service_id"`
		RequestedByUserID *string `json:"requested_by_user_id"`
		Operation         string  `json:"operation"`
		Status            string  `json:"status"`
		ErrorMessage      *string `json:"error_message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.ServiceID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("service_id = %q, want %q", body.ServiceID, "11111111-1111-1111-1111-111111111111")
	}
	if body.Operation != "Provision" {
		t.Errorf("operation = %q, want %q", body.Operation, "Provision")
	}
	if body.Status != "Pending" {
		t.Errorf("status = %q, want %q", body.Status, "Pending")
	}
	// goal 8/10: nullable fields serialize as JSON null when unset, not
	// omitted and not a zero value like "" or 0001-01-01.
	if body.RequestedByUserID != nil {
		t.Errorf("requested_by_user_id = %v, want null (no authenticated caller in this test)", *body.RequestedByUserID)
	}
	if body.ErrorMessage != nil {
		t.Errorf("error_message = %v, want null", *body.ErrorMessage)
	}
}

func TestProvisioningHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeProvisioningService())

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProvisioningHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeProvisioningService()
	svc.err = apperror.Invalid("operation: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs", strings.NewReader(`{"operation":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestProvisioningHandlerCreatePropagatesConflictOnUnknownService(t *testing.T) {
	svc := newFakeProvisioningService()
	svc.err = apperror.Conflict("create provisioning job: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestProvisioningHandlerList(t *testing.T) {
	a := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusPending}
	b := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationSuspend, Status: provisioning.ProvisioningStatusPending}
	router := newTestRouter(newFakeProvisioningService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		ProvisioningJobs []struct {
			ID string `json:"id"`
		} `json:"provisioning_jobs"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.ProvisioningJobs) != 2 {
		t.Fatalf("len(provisioning_jobs) = %d, want 2", len(body.ProvisioningJobs))
	}
}

// TestProvisioningHandlerListFiltersByServiceIDQueryParameter proves the
// ?service_id= filter dispatches to ListByServiceID and returns only
// that Service's jobs (goal 10's requirement, exercised through the HTTP
// layer).
func TestProvisioningHandlerListFiltersByServiceIDQueryParameter(t *testing.T) {
	targetService := uuid.New()
	forTarget := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: targetService, Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusPending}
	other := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusPending}
	router := newTestRouter(newFakeProvisioningService(forTarget, other))

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs?service_id="+targetService.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		ProvisioningJobs []struct {
			ID        string `json:"id"`
			ServiceID string `json:"service_id"`
		} `json:"provisioning_jobs"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.ProvisioningJobs) != 1 {
		t.Fatalf("len(provisioning_jobs) = %d, want 1; got %+v", len(body.ProvisioningJobs), body.ProvisioningJobs)
	}
	if body.ProvisioningJobs[0].ID != forTarget.ID.String() {
		t.Errorf("returned job id = %q, want %q", body.ProvisioningJobs[0].ID, forTarget.ID.String())
	}
}

func TestProvisioningHandlerListRejectsMalformedServiceIDQueryParameter(t *testing.T) {
	router := newTestRouter(newFakeProvisioningService())

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs?service_id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProvisioningHandlerGet(t *testing.T) {
	j := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusPending}
	router := newTestRouter(newFakeProvisioningService(j))

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs/"+j.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProvisioningHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeProvisioningService())

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProvisioningHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeProvisioningService())

	req := httptest.NewRequest(http.MethodGet, "/provisioning-jobs/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProvisioningHandlerDelete(t *testing.T) {
	j := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusPending}
	router := newTestRouter(newFakeProvisioningService(j))

	req := httptest.NewRequest(http.MethodDelete, "/provisioning-jobs/"+j.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestProvisioningHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeProvisioningService())

	req := httptest.NewRequest(http.MethodDelete, "/provisioning-jobs/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProvisioningHandlerStart(t *testing.T) {
	j := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusPending}
	router := newTestRouter(newFakeProvisioningService(j))

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/"+j.ID.String()+"/start", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "Running" {
		t.Errorf("status = %q, want %q", body.Status, "Running")
	}
}

// TestProvisioningHandlerStartPropagatesInvalidTransitionConflict proves
// an invalid-transition Conflict from ProvisioningService reaches the
// client as 409 — this handler has no special-case knowledge of the
// state machine, it just translates whatever error the service layer
// returns (goal 10: "invalid state transitions fail").
func TestProvisioningHandlerStartPropagatesInvalidTransitionConflict(t *testing.T) {
	svc := newFakeProvisioningService()
	svc.err = apperror.Conflict("cannot transition provisioning job from Succeeded to Running")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/"+uuid.New().String()+"/start", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestProvisioningHandlerSucceed(t *testing.T) {
	j := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusRunning}
	router := newTestRouter(newFakeProvisioningService(j))

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/"+j.ID.String()+"/succeed", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProvisioningHandlerFail(t *testing.T) {
	j := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusRunning}
	router := newTestRouter(newFakeProvisioningService(j))

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/"+j.ID.String()+"/fail",
		strings.NewReader(`{"error_message":"device unreachable"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "Failed" {
		t.Errorf("status = %q, want %q", body.Status, "Failed")
	}
	if body.ErrorMessage != "device unreachable" {
		t.Errorf("error_message = %q, want %q", body.ErrorMessage, "device unreachable")
	}
}

func TestProvisioningHandlerCancel(t *testing.T) {
	j := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusPending}
	router := newTestRouter(newFakeProvisioningService(j))

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/"+j.ID.String()+"/cancel", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProvisioningHandlerRetry(t *testing.T) {
	j := provisioning.ProvisioningJob{ID: uuid.New(), ServiceID: uuid.New(), Operation: provisioning.ProvisioningOperationProvision, Status: provisioning.ProvisioningStatusFailed, RetryCount: 1}
	router := newTestRouter(newFakeProvisioningService(j))

	req := httptest.NewRequest(http.MethodPost, "/provisioning-jobs/"+j.ID.String()+"/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Status     string `json:"status"`
		RetryCount int    `json:"retry_count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "Pending" {
		t.Errorf("status = %q, want %q", body.Status, "Pending")
	}
	if body.RetryCount != 2 {
		t.Errorf("retry_count = %d, want %d", body.RetryCount, 2)
	}
}
