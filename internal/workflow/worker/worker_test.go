package worker_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow/worker"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakePendingSource struct {
	mu        sync.Mutex
	instances []workflow.Instance
	calls     int
}

func (f *fakePendingSource) NextPending(context.Context) (workflow.Instance, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if len(f.instances) == 0 {
		return workflow.Instance{}, false, nil
	}
	next := f.instances[0]
	f.instances = f.instances[1:]
	return next, true, nil
}

type fakeEngine struct {
	mu       sync.Mutex
	executed []uuid.UUID
	err      error
}

func (f *fakeEngine) Execute(_ context.Context, instanceID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.executed = append(f.executed, instanceID)
	return f.err
}

func (f *fakeEngine) executedIDs() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uuid.UUID(nil), f.executed...)
}

func TestWorkerExecutesAFoundPendingInstance(t *testing.T) {
	instanceID := uuid.New()
	source := &fakePendingSource{instances: []workflow.Instance{{ID: instanceID}}}
	engine := &fakeEngine{}
	w := worker.New(source, engine, time.Millisecond, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	waitFor(t, func() bool { return len(engine.executedIDs()) == 1 })
	cancel()
	waitForDone(t, done)

	if got := engine.executedIDs(); len(got) != 1 || got[0] != instanceID {
		t.Fatalf("executed = %v, want [%v]", got, instanceID)
	}
}

func TestWorkerDoesNothingWhenNoneArePending(t *testing.T) {
	source := &fakePendingSource{}
	engine := &fakeEngine{}
	w := worker.New(source, engine, time.Millisecond, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// Let it poll a handful of times, finding nothing pending each time.
	time.Sleep(20 * time.Millisecond)
	cancel()
	waitForDone(t, done)

	if got := engine.executedIDs(); len(got) != 0 {
		t.Fatalf("executed = %v, want none", got)
	}
}

func TestWorkerStopsPromptlyOnContextCancellation(t *testing.T) {
	source := &fakePendingSource{}
	engine := &fakeEngine{}
	// A long poll interval: if Run did not check ctx.Done() promptly, this
	// test would need to wait out the whole interval to see it stop.
	w := worker.New(source, engine, time.Minute, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// Give Run a moment to enter its first sleep, then cancel.
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop promptly after context cancellation")
	}
}

func TestWorkerLogsAndContinuesAfterAnExecuteFailure(t *testing.T) {
	instanceID := uuid.New()
	source := &fakePendingSource{instances: []workflow.Instance{{ID: instanceID}}}
	engine := &fakeEngine{err: errors.New("plugin exploded")}
	w := worker.New(source, engine, time.Millisecond, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	waitFor(t, func() bool { return len(engine.executedIDs()) == 1 })
	cancel()
	waitForDone(t, done)
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		if condition() {
			return
		}
		select {
		case <-deadline:
			t.Fatal("condition not met before deadline")
		case <-time.After(time.Millisecond):
		}
	}
}

func waitForDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}
