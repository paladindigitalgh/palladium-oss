package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow/service"
)

type fakeRepo struct {
	instances map[uuid.UUID]workflow.Instance
}

func newFakeRepo() *fakeRepo { return &fakeRepo{instances: map[uuid.UUID]workflow.Instance{}} }

func (r *fakeRepo) Get(_ context.Context, id uuid.UUID) (workflow.Instance, error) {
	i, ok := r.instances[id]
	if !ok {
		return workflow.Instance{}, apperror.NotFound("workflow instance not found")
	}
	return i, nil
}
func (r *fakeRepo) List(context.Context) ([]workflow.Instance, error) { return nil, nil }
func (r *fakeRepo) ListByServiceID(context.Context, uuid.UUID) ([]workflow.Instance, error) {
	return nil, nil
}
func (r *fakeRepo) Create(_ context.Context, i workflow.Instance) (workflow.Instance, error) {
	i.ID = uuid.New()
	r.instances[i.ID] = i
	return i, nil
}
func (r *fakeRepo) Update(_ context.Context, i workflow.Instance) (workflow.Instance, error) {
	r.instances[i.ID] = i
	return i, nil
}
func (r *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.instances, id)
	return nil
}

type fakeEvents struct {
	created []event.Event
}

func (e *fakeEvents) Create(_ context.Context, ev event.Event) (event.Event, error) {
	ev.ID = uuid.New()
	e.created = append(e.created, ev)
	return ev, nil
}
func (e *fakeEvents) ListByEntity(context.Context, string, uuid.UUID) ([]event.Event, error) {
	return e.created, nil
}

func (e *fakeEvents) ListRecent(context.Context, int) ([]event.Event, error) {
	return e.created, nil
}

func TestCreateForcesInitialState(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo, &fakeEvents{}, clock.NewFrozen(time.Now()))

	created, err := svc.Create(context.Background(), workflow.Instance{
		ServiceID:      uuid.New(),
		DefinitionName: "suspend-service",
		Status:         workflow.StatusSucceeded, // must be ignored
		RetryCount:     7,                        // must be ignored
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != workflow.StatusPending {
		t.Errorf("Status = %s, want %s", created.Status, workflow.StatusPending)
	}
	if created.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", created.RetryCount)
	}
}

func TestStartTransitionsAndEmitsEvent(t *testing.T) {
	repo := newFakeRepo()
	events := &fakeEvents{}
	svc := service.New(repo, events, clock.NewFrozen(time.Now()))

	created, _ := svc.Create(context.Background(), workflow.Instance{ServiceID: uuid.New(), DefinitionName: "suspend-service"})

	started, err := svc.Start(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if started.Status != workflow.StatusRunning {
		t.Errorf("Status = %s, want %s", started.Status, workflow.StatusRunning)
	}
	if started.StartedAt == nil {
		t.Error("StartedAt = nil, want a timestamp")
	}
	if len(events.created) != 1 || events.created[0].Type != "workflow.started" {
		t.Fatalf("events = %+v, want one workflow.started event", events.created)
	}
	if events.created[0].EntityType != "service" || events.created[0].EntityID != started.ServiceID {
		t.Errorf("event entity = %s/%s, want service/%s", events.created[0].EntityType, events.created[0].EntityID, started.ServiceID)
	}
}

func TestStartRejectsNonPendingInstance(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo, &fakeEvents{}, clock.NewFrozen(time.Now()))

	created, _ := svc.Create(context.Background(), workflow.Instance{ServiceID: uuid.New(), DefinitionName: "suspend-service"})
	if _, err := svc.Start(context.Background(), created.ID); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}

	_, err := svc.Start(context.Background(), created.ID)
	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("second Start() error kind = %v, want %v", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestRetryIncrementsRetryCount(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo, &fakeEvents{}, clock.NewFrozen(time.Now()))

	created, _ := svc.Create(context.Background(), workflow.Instance{ServiceID: uuid.New(), DefinitionName: "suspend-service"})
	svc.Start(context.Background(), created.ID)
	failed, _ := svc.Fail(context.Background(), created.ID, "device unreachable")
	if failed.RetryCount != 0 {
		t.Fatalf("RetryCount after Fail = %d, want 0", failed.RetryCount)
	}

	retried, err := svc.Retry(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if retried.RetryCount != 1 {
		t.Errorf("RetryCount after Retry = %d, want 1", retried.RetryCount)
	}
	if retried.Status != workflow.StatusPending {
		t.Errorf("Status after Retry = %s, want %s", retried.Status, workflow.StatusPending)
	}
}
