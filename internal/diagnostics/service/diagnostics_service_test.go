package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeRegistry is an in-memory diagnostics.Registry. Like every other
// domain's fake repository in this codebase, it exists so
// DiagnosticsService's logic — locate, then delegate — is tested without
// depending on diagnostics.DefaultRegistry's own implementation;
// internal/diagnostics/registry_test.go already covers that directly. It
// tracks whether Get was actually called, which is what lets
// TestDiagnosticsServiceRunPropagatesNotFound prove the service reports
// an unregistered name rather than silently succeeding.
type fakeRegistry struct {
	byName    map[string]diagnostics.Diagnostic
	getCalled bool
}

func newFakeRegistry(diags ...diagnostics.Diagnostic) *fakeRegistry {
	f := &fakeRegistry{byName: make(map[string]diagnostics.Diagnostic)}
	for _, d := range diags {
		f.byName[d.Name()] = d
	}
	return f
}

func (f *fakeRegistry) Register(d diagnostics.Diagnostic) {
	f.byName[d.Name()] = d
}

func (f *fakeRegistry) Get(name string) (diagnostics.Diagnostic, bool) {
	f.getCalled = true
	d, ok := f.byName[name]
	return d, ok
}

var _ diagnostics.Registry = (*fakeRegistry)(nil)

// fakeDiagnostic lets tests observe exactly what Request the service
// passed through, and control what Run returns, without depending on
// diagnostics.BasicONUCheck's own fixed placeholder behavior.
type fakeDiagnostic struct {
	name      string
	result    *diagnostics.Result
	err       error
	lastReq   diagnostics.Request
	runCalled bool
}

func (d *fakeDiagnostic) Name() string { return d.name }
func (d *fakeDiagnostic) Run(_ context.Context, request diagnostics.Request) (*diagnostics.Result, error) {
	d.runCalled = true
	d.lastReq = request
	if d.err != nil {
		return nil, d.err
	}
	return d.result, nil
}

var _ diagnostics.Diagnostic = (*fakeDiagnostic)(nil)

func TestDiagnosticsServiceRunDelegatesToRegisteredDiagnostic(t *testing.T) {
	want := &diagnostics.Result{Name: "Test Diagnostic"}
	diag := &fakeDiagnostic{name: "Test Diagnostic", result: want}
	registry := newFakeRegistry(diag)
	svc := service.NewDiagnosticsService(registry)

	req := diagnostics.Request{ONUID: uuid.New()}
	got, err := svc.Run(context.Background(), "Test Diagnostic", req)
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if got != want {
		t.Errorf("Run() = %+v, want the exact Result the Diagnostic returned", got)
	}
	if !diag.runCalled {
		t.Error("the registered Diagnostic's Run() was never called")
	}
	if diag.lastReq != req {
		t.Errorf("Diagnostic.Run() received %+v, want %+v", diag.lastReq, req)
	}
}

func TestDiagnosticsServiceRunPropagatesNotFound(t *testing.T) {
	registry := newFakeRegistry() // nothing registered
	svc := service.NewDiagnosticsService(registry)

	_, err := svc.Run(context.Background(), "Unregistered Diagnostic", diagnostics.Request{})

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
	if !registry.getCalled {
		t.Error("registry Get() was never called")
	}
}

func TestDiagnosticsServiceRunPropagatesDiagnosticError(t *testing.T) {
	wantErr := apperror.Internal("run diagnostic", nil)
	diag := &fakeDiagnostic{name: "Failing Diagnostic", err: wantErr}
	registry := newFakeRegistry(diag)
	svc := service.NewDiagnosticsService(registry)

	_, err := svc.Run(context.Background(), "Failing Diagnostic", diagnostics.Request{})

	if err != wantErr {
		t.Errorf("Run() error = %v, want the exact error the Diagnostic returned", err)
	}
}
