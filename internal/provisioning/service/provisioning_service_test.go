package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/service"
)

// fakeProvisioningRepository is an in-memory
// provisioning.ProvisioningRepository. Like
// internal/serviceequipment/service/service_equipment_service_test.go's
// fakeServiceEquipmentRepository, it exists so ProvisioningService's
// business logic — validate, enforce state transitions, then delegate —
// is tested without a real database; internal/provisioning/postgres's own
// tests already cover the repository itself against real PostgreSQL.
type fakeProvisioningRepository struct {
	byID         map[uuid.UUID]provisioning.ProvisioningJob
	createCalled bool
	updateCalled bool
}

func newFakeProvisioningRepository(jobs ...provisioning.ProvisioningJob) *fakeProvisioningRepository {
	f := &fakeProvisioningRepository{byID: make(map[uuid.UUID]provisioning.ProvisioningJob)}
	for _, j := range jobs {
		f.byID[j.ID] = j
	}
	return f
}

func (f *fakeProvisioningRepository) Get(_ context.Context, id uuid.UUID) (provisioning.ProvisioningJob, error) {
	j, ok := f.byID[id]
	if !ok {
		return provisioning.ProvisioningJob{}, apperror.NotFound("provisioning job not found")
	}
	return j, nil
}

func (f *fakeProvisioningRepository) List(_ context.Context) ([]provisioning.ProvisioningJob, error) {
	jobs := make([]provisioning.ProvisioningJob, 0, len(f.byID))
	for _, j := range f.byID {
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (f *fakeProvisioningRepository) ListByServiceID(_ context.Context, serviceID uuid.UUID) ([]provisioning.ProvisioningJob, error) {
	var jobs []provisioning.ProvisioningJob
	for _, j := range f.byID {
		if j.ServiceID == serviceID {
			jobs = append(jobs, j)
		}
	}
	return jobs, nil
}

func (f *fakeProvisioningRepository) Create(_ context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	f.createCalled = true
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	f.byID[j.ID] = j
	return j, nil
}

func (f *fakeProvisioningRepository) Update(_ context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	f.updateCalled = true
	if _, ok := f.byID[j.ID]; !ok {
		return provisioning.ProvisioningJob{}, apperror.NotFound("provisioning job not found")
	}
	f.byID[j.ID] = j
	return j, nil
}

func (f *fakeProvisioningRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("provisioning job not found")
	}
	delete(f.byID, id)
	return nil
}

var _ provisioning.ProvisioningRepository = (*fakeProvisioningRepository)(nil)

var testNow = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func validProvisioningJob() provisioning.ProvisioningJob {
	return provisioning.ProvisioningJob{
		ServiceID: uuid.New(),
		Operation: provisioning.ProvisioningOperationProvision,
		Status:    provisioning.ProvisioningStatusPending,
	}
}

func TestProvisioningServiceCreateSucceeds(t *testing.T) {
	repo := newFakeProvisioningRepository()
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	created, err := svc.Create(context.Background(), validProvisioningJob())
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if !repo.createCalled {
		t.Error("repository Create() was never called")
	}
}

// TestProvisioningServiceCreateForcesInitialLifecycleState proves Create
// ignores caller-supplied Status/RetryCount/ErrorMessage/StartedAt/
// CompletedAt and always starts a job at Pending with zeroed lifecycle
// fields.
func TestProvisioningServiceCreateForcesInitialLifecycleState(t *testing.T) {
	repo := newFakeProvisioningRepository()
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	bogusRetryCount := 7
	bogusErr := "should not survive"
	bogusTime := testNow.Add(-time.Hour)

	suspicious := validProvisioningJob()
	suspicious.Status = provisioning.ProvisioningStatusSucceeded
	suspicious.RetryCount = bogusRetryCount
	suspicious.ErrorMessage = &bogusErr
	suspicious.StartedAt = &bogusTime
	suspicious.CompletedAt = &bogusTime

	created, err := svc.Create(context.Background(), suspicious)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.Status != provisioning.ProvisioningStatusPending {
		t.Errorf("Status = %q, want %q", created.Status, provisioning.ProvisioningStatusPending)
	}
	if created.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", created.RetryCount)
	}
	if created.ErrorMessage != nil || created.StartedAt != nil || created.CompletedAt != nil {
		t.Errorf("lifecycle fields = (%v, %v, %v), want all nil", created.ErrorMessage, created.StartedAt, created.CompletedAt)
	}
}

func TestProvisioningServiceCreateRejectsInvalidJobWithoutPersisting(t *testing.T) {
	repo := newFakeProvisioningRepository()
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	_, err := svc.Create(context.Background(), provisioning.ProvisioningJob{}) // no ServiceID, Operation

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestProvisioningServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeProvisioningRepository()
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProvisioningServiceListDelegatesToRepository(t *testing.T) {
	a := validProvisioningJob()
	a.ID = uuid.New()
	b := validProvisioningJob()
	b.ID = uuid.New()
	repo := newFakeProvisioningRepository(a, b)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	jobs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(jobs))
	}
}

// TestProvisioningServiceListByServiceIDReturnsOnlyThatServicesJobs
// mirrors the same explicit requirement (goal 10) already proven at the
// repository layer, one layer up.
func TestProvisioningServiceListByServiceIDReturnsOnlyThatServicesJobs(t *testing.T) {
	serviceID := uuid.New()
	forService := validProvisioningJob()
	forService.ID = uuid.New()
	forService.ServiceID = serviceID
	other := validProvisioningJob()
	other.ID = uuid.New()
	repo := newFakeProvisioningRepository(forService, other)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	jobs, err := svc.ListByServiceID(context.Background(), serviceID)
	if err != nil {
		t.Fatalf("ListByServiceID() = %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != forService.ID {
		t.Errorf("ListByServiceID() = %+v, want only %+v", jobs, forService)
	}
}

func TestProvisioningServiceDeleteSucceeds(t *testing.T) {
	existing := validProvisioningJob()
	existing.ID = uuid.New()
	repo := newFakeProvisioningRepository(existing)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProvisioningServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeProvisioningRepository()
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

// -- State transitions -------------------------------------------------
//
// These tests are goal 10's explicit requirement: "Valid state
// transitions succeed. Invalid state transitions fail."

func TestProvisioningServiceStartTransitionsPendingToRunningAndStampsStartedAt(t *testing.T) {
	existing := validProvisioningJob()
	existing.ID = uuid.New()
	existing.Status = provisioning.ProvisioningStatusPending
	repo := newFakeProvisioningRepository(existing)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	updated, err := svc.Start(context.Background(), existing.ID)
	if err != nil {
		t.Fatalf("Start() = %v", err)
	}
	if updated.Status != provisioning.ProvisioningStatusRunning {
		t.Errorf("Status = %q, want %q", updated.Status, provisioning.ProvisioningStatusRunning)
	}
	if updated.StartedAt == nil || !updated.StartedAt.Equal(testNow) {
		t.Errorf("StartedAt = %v, want %v", updated.StartedAt, testNow)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestProvisioningServiceStartRejectsFromNonPendingStatus(t *testing.T) {
	for _, from := range []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusRunning,
		provisioning.ProvisioningStatusSucceeded,
		provisioning.ProvisioningStatusFailed,
		provisioning.ProvisioningStatusCancelled,
	} {
		existing := validProvisioningJob()
		existing.ID = uuid.New()
		existing.Status = from
		repo := newFakeProvisioningRepository(existing)
		svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

		_, err := svc.Start(context.Background(), existing.ID)

		if !apperror.Is(err, apperror.KindConflict) {
			t.Errorf("Start() from %q: Kind = %q, want %q", from, apperror.KindOf(err), apperror.KindConflict)
		}
		if repo.updateCalled {
			t.Errorf("Start() from %q: repository Update() was called despite an invalid transition", from)
		}
	}
}

func TestProvisioningServiceSucceedTransitionsRunningToSucceededAndStampsCompletedAt(t *testing.T) {
	existing := validProvisioningJob()
	existing.ID = uuid.New()
	existing.Status = provisioning.ProvisioningStatusRunning
	repo := newFakeProvisioningRepository(existing)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	updated, err := svc.Succeed(context.Background(), existing.ID)
	if err != nil {
		t.Fatalf("Succeed() = %v", err)
	}
	if updated.Status != provisioning.ProvisioningStatusSucceeded {
		t.Errorf("Status = %q, want %q", updated.Status, provisioning.ProvisioningStatusSucceeded)
	}
	if updated.CompletedAt == nil || !updated.CompletedAt.Equal(testNow) {
		t.Errorf("CompletedAt = %v, want %v", updated.CompletedAt, testNow)
	}
}

func TestProvisioningServiceSucceedRejectsFromPending(t *testing.T) {
	existing := validProvisioningJob()
	existing.ID = uuid.New()
	existing.Status = provisioning.ProvisioningStatusPending
	repo := newFakeProvisioningRepository(existing)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	_, err := svc.Succeed(context.Background(), existing.ID)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestProvisioningServiceFailTransitionsRunningToFailedAndRecordsErrorMessage(t *testing.T) {
	existing := validProvisioningJob()
	existing.ID = uuid.New()
	existing.Status = provisioning.ProvisioningStatusRunning
	repo := newFakeProvisioningRepository(existing)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	updated, err := svc.Fail(context.Background(), existing.ID, "device unreachable")
	if err != nil {
		t.Fatalf("Fail() = %v", err)
	}
	if updated.Status != provisioning.ProvisioningStatusFailed {
		t.Errorf("Status = %q, want %q", updated.Status, provisioning.ProvisioningStatusFailed)
	}
	if updated.ErrorMessage == nil || *updated.ErrorMessage != "device unreachable" {
		t.Errorf("ErrorMessage = %v, want %q", updated.ErrorMessage, "device unreachable")
	}
	if updated.CompletedAt == nil || !updated.CompletedAt.Equal(testNow) {
		t.Errorf("CompletedAt = %v, want %v", updated.CompletedAt, testNow)
	}
}

func TestProvisioningServiceFailRejectsFromPending(t *testing.T) {
	existing := validProvisioningJob()
	existing.ID = uuid.New()
	existing.Status = provisioning.ProvisioningStatusPending
	repo := newFakeProvisioningRepository(existing)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	_, err := svc.Fail(context.Background(), existing.ID, "should not apply")

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestProvisioningServiceCancelAllowedFromPendingAndRunning(t *testing.T) {
	for _, from := range []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusPending,
		provisioning.ProvisioningStatusRunning,
	} {
		existing := validProvisioningJob()
		existing.ID = uuid.New()
		existing.Status = from
		repo := newFakeProvisioningRepository(existing)
		svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

		updated, err := svc.Cancel(context.Background(), existing.ID)
		if err != nil {
			t.Fatalf("Cancel() from %q = %v", from, err)
		}
		if updated.Status != provisioning.ProvisioningStatusCancelled {
			t.Errorf("Cancel() from %q: Status = %q, want %q", from, updated.Status, provisioning.ProvisioningStatusCancelled)
		}
		if updated.CompletedAt == nil || !updated.CompletedAt.Equal(testNow) {
			t.Errorf("Cancel() from %q: CompletedAt = %v, want %v", from, updated.CompletedAt, testNow)
		}
	}
}

func TestProvisioningServiceCancelRejectsFromTerminalStatuses(t *testing.T) {
	for _, from := range []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusSucceeded,
		provisioning.ProvisioningStatusFailed,
		provisioning.ProvisioningStatusCancelled,
	} {
		existing := validProvisioningJob()
		existing.ID = uuid.New()
		existing.Status = from
		repo := newFakeProvisioningRepository(existing)
		svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

		_, err := svc.Cancel(context.Background(), existing.ID)

		if !apperror.Is(err, apperror.KindConflict) {
			t.Errorf("Cancel() from %q: Kind = %q, want %q", from, apperror.KindOf(err), apperror.KindConflict)
		}
	}
}

// TestProvisioningServiceRetryTransitionsFailedToPendingAndIncrementsRetryCount
// is goal 10's explicit requirement that RetryCount round-trips, applied
// to the one transition that actually changes it.
func TestProvisioningServiceRetryTransitionsFailedToPendingAndIncrementsRetryCount(t *testing.T) {
	existing := validProvisioningJob()
	existing.ID = uuid.New()
	existing.Status = provisioning.ProvisioningStatusFailed
	existing.RetryCount = 2
	repo := newFakeProvisioningRepository(existing)
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	updated, err := svc.Retry(context.Background(), existing.ID)
	if err != nil {
		t.Fatalf("Retry() = %v", err)
	}
	if updated.Status != provisioning.ProvisioningStatusPending {
		t.Errorf("Status = %q, want %q", updated.Status, provisioning.ProvisioningStatusPending)
	}
	if updated.RetryCount != 3 {
		t.Errorf("RetryCount = %d, want %d", updated.RetryCount, 3)
	}
}

func TestProvisioningServiceRetryRejectsFromNonFailedStatus(t *testing.T) {
	for _, from := range []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusPending,
		provisioning.ProvisioningStatusRunning,
		provisioning.ProvisioningStatusSucceeded,
		provisioning.ProvisioningStatusCancelled,
	} {
		existing := validProvisioningJob()
		existing.ID = uuid.New()
		existing.Status = from
		repo := newFakeProvisioningRepository(existing)
		svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

		_, err := svc.Retry(context.Background(), existing.ID)

		if !apperror.Is(err, apperror.KindConflict) {
			t.Errorf("Retry() from %q: Kind = %q, want %q", from, apperror.KindOf(err), apperror.KindConflict)
		}
	}
}

func TestProvisioningServiceTransitionPropagatesNotFound(t *testing.T) {
	repo := newFakeProvisioningRepository()
	svc := service.NewProvisioningService(repo, clock.NewFrozen(testNow))

	_, err := svc.Start(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
