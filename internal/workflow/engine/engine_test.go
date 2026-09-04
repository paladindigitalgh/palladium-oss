package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/plugin"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow/engine"
)

type fakeTransitioner struct {
	instance  workflow.Instance
	started   bool
	succeeded bool
	failed    bool
	failMsg   string
}

func (f *fakeTransitioner) Start(context.Context, uuid.UUID) (workflow.Instance, error) {
	f.started = true
	f.instance.Status = workflow.StatusRunning
	return f.instance, nil
}
func (f *fakeTransitioner) Succeed(context.Context, uuid.UUID) (workflow.Instance, error) {
	f.succeeded = true
	f.instance.Status = workflow.StatusSucceeded
	return f.instance, nil
}
func (f *fakeTransitioner) Fail(_ context.Context, _ uuid.UUID, msg string) (workflow.Instance, error) {
	f.failed = true
	f.failMsg = msg
	f.instance.Status = workflow.StatusFailed
	return f.instance, nil
}

type fakeServiceRepo struct {
	svc     service.Service
	updated *service.Service
}

func (f *fakeServiceRepo) Get(context.Context, uuid.UUID) (service.Service, error) { return f.svc, nil }
func (f *fakeServiceRepo) List(context.Context) ([]service.Service, error)         { return nil, nil }
func (f *fakeServiceRepo) Create(_ context.Context, s service.Service) (service.Service, error) {
	return s, nil
}
func (f *fakeServiceRepo) Update(_ context.Context, s service.Service) (service.Service, error) {
	f.updated = &s
	return s, nil
}
func (f *fakeServiceRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (f *fakeServiceRepo) ListByLocationID(context.Context, uuid.UUID) ([]service.Service, error) {
	return nil, nil
}

type fakeEquipmentRepo struct {
	active []serviceequipment.ServiceEquipment
	err    error
}

func (f fakeEquipmentRepo) Get(context.Context, uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	return serviceequipment.ServiceEquipment{}, nil
}
func (f fakeEquipmentRepo) List(context.Context) ([]serviceequipment.ServiceEquipment, error) {
	return nil, nil
}
func (f fakeEquipmentRepo) Create(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	return e, nil
}
func (f fakeEquipmentRepo) Update(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	return e, nil
}
func (f fakeEquipmentRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (f fakeEquipmentRepo) GetActiveByDeviceID(context.Context, uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	return serviceequipment.ServiceEquipment{}, nil
}
func (f fakeEquipmentRepo) ListActiveByServiceID(context.Context, uuid.UUID) ([]serviceequipment.ServiceEquipment, error) {
	return f.active, f.err
}

type fakePlugin struct {
	name string
	err  error
}

func (p fakePlugin) Name() string   { return p.name }
func (p fakePlugin) Vendor() string { return "fake" }
func (p fakePlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{plugin.SuspendService}
}
func (p fakePlugin) Execute(context.Context, plugin.Capability, plugin.Resource) (plugin.Result, error) {
	if p.err != nil {
		return plugin.Result{}, p.err
	}
	return plugin.Result{Message: "ok"}, nil
}

func TestExecuteSucceedsWhenEveryPluginCallSucceeds(t *testing.T) {
	instanceID := uuid.New()
	serviceID := uuid.New()

	transitioner := &fakeTransitioner{instance: workflow.Instance{ID: instanceID, ServiceID: serviceID, DefinitionName: "suspend-service", Status: workflow.StatusPending}}
	equipment := fakeEquipmentRepo{active: []serviceequipment.ServiceEquipment{{ID: uuid.New(), Role: "ONU"}}}
	registry := plugin.NewDefaultRegistry()
	registry.Register(fakePlugin{name: "fake"})

	e := engine.NewDefaultEngine(transitioner, &fakeServiceRepo{svc: service.Service{ID: serviceID}}, equipment, registry, clock.New())

	if err := e.Execute(context.Background(), instanceID); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}
	if !transitioner.started || !transitioner.succeeded {
		t.Fatalf("Execute() started=%v succeeded=%v, want both true", transitioner.started, transitioner.succeeded)
	}
}

func TestExecuteUpdatesServiceStatusOnSuccess(t *testing.T) {
	instanceID := uuid.New()
	serviceID := uuid.New()

	transitioner := &fakeTransitioner{instance: workflow.Instance{ID: instanceID, ServiceID: serviceID, DefinitionName: "suspend-service", Status: workflow.StatusPending}}
	equipment := fakeEquipmentRepo{active: []serviceequipment.ServiceEquipment{{ID: uuid.New(), Role: "ONU"}}}
	registry := plugin.NewDefaultRegistry()
	registry.Register(fakePlugin{name: "fake"})
	services := &fakeServiceRepo{svc: service.Service{ID: serviceID, Status: service.ServiceStatusActive}}

	e := engine.NewDefaultEngine(transitioner, services, equipment, registry, clock.New())

	if err := e.Execute(context.Background(), instanceID); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}
	if services.updated == nil {
		t.Fatal("Execute() never called Service.Update")
	}
	if services.updated.Status != service.ServiceStatusSuspended {
		t.Errorf("Service.Status = %s, want %s", services.updated.Status, service.ServiceStatusSuspended)
	}
	if services.updated.SuspendedAt == nil {
		t.Error("Service.SuspendedAt = nil, want a timestamp")
	}
}

func TestExecuteFailsWhenNoPluginRegisteredForCapability(t *testing.T) {
	instanceID := uuid.New()
	serviceID := uuid.New()

	transitioner := &fakeTransitioner{instance: workflow.Instance{ID: instanceID, ServiceID: serviceID, DefinitionName: "suspend-service", Status: workflow.StatusPending}}
	equipment := fakeEquipmentRepo{active: []serviceequipment.ServiceEquipment{{ID: uuid.New(), Role: "ONU"}}}
	registry := plugin.NewDefaultRegistry() // nothing registered

	e := engine.NewDefaultEngine(transitioner, &fakeServiceRepo{svc: service.Service{ID: serviceID}}, equipment, registry, clock.New())

	if err := e.Execute(context.Background(), instanceID); err == nil {
		t.Fatal("Execute() = nil, want an error")
	}
	if !transitioner.failed {
		t.Fatal("Execute() did not transition the instance to Failed")
	}
}

func TestExecutePropagatesPluginError(t *testing.T) {
	instanceID := uuid.New()
	serviceID := uuid.New()

	transitioner := &fakeTransitioner{instance: workflow.Instance{ID: instanceID, ServiceID: serviceID, DefinitionName: "suspend-service", Status: workflow.StatusPending}}
	equipment := fakeEquipmentRepo{active: []serviceequipment.ServiceEquipment{{ID: uuid.New(), Role: "ONU"}}}
	registry := plugin.NewDefaultRegistry()
	registry.Register(fakePlugin{name: "fake", err: errors.New("device unreachable")})

	e := engine.NewDefaultEngine(transitioner, &fakeServiceRepo{svc: service.Service{ID: serviceID}}, equipment, registry, clock.New())

	err := e.Execute(context.Background(), instanceID)
	if err == nil {
		t.Fatal("Execute() = nil, want an error")
	}
	if !transitioner.failed {
		t.Fatal("Execute() did not transition the instance to Failed")
	}
}

func TestExecuteDoesNotTouchServiceOrEquipmentWhenNotPending(t *testing.T) {
	// Start itself is responsible for rejecting a non-Pending instance
	// (see internal/workflow/service.Service.transition); this test only
	// confirms Execute propagates that error unchanged rather than
	// swallowing it.
	instanceID := uuid.New()
	transitioner := &conflictTransitioner{}
	equipment := fakeEquipmentRepo{}
	registry := plugin.NewDefaultRegistry()

	e := engine.NewDefaultEngine(transitioner, &fakeServiceRepo{}, equipment, registry, clock.New())

	err := e.Execute(context.Background(), instanceID)
	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Execute() error kind = %v, want %v", apperror.KindOf(err), apperror.KindConflict)
	}
}

type conflictTransitioner struct{}

func (conflictTransitioner) Start(context.Context, uuid.UUID) (workflow.Instance, error) {
	return workflow.Instance{}, apperror.Conflict("cannot transition workflow instance from Succeeded to Running")
}
func (conflictTransitioner) Succeed(context.Context, uuid.UUID) (workflow.Instance, error) {
	return workflow.Instance{}, nil
}
func (conflictTransitioner) Fail(context.Context, uuid.UUID, string) (workflow.Instance, error) {
	return workflow.Instance{}, nil
}
