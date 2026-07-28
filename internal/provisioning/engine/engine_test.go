package engine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/connectors"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/engine"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// -- fakes ---------------------------------------------------------------
//
// One fake per injected repository, each scoped to exactly what these
// tests need to observe or force: fakeProvisioningRepository records
// every Update it sees (so tests can assert the exact Pending -> Running
// -> Succeeded/Failed sequence DefaultEngine.Execute is supposed to
// persist), and fakeServiceRepository/fakeServiceEquipmentRepository each
// carry a settable err so a test can simulate that one specific
// dependency failing without touching the others.

type fakeProvisioningRepository struct {
	byID        map[uuid.UUID]provisioning.ProvisioningJob
	updateCalls []provisioning.ProvisioningJob
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

func (f *fakeProvisioningRepository) List(context.Context) ([]provisioning.ProvisioningJob, error) {
	panic("not used by these tests")
}

func (f *fakeProvisioningRepository) ListByServiceID(context.Context, uuid.UUID) ([]provisioning.ProvisioningJob, error) {
	panic("not used by these tests")
}

func (f *fakeProvisioningRepository) Create(context.Context, provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	panic("not used by these tests")
}

func (f *fakeProvisioningRepository) Update(_ context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	if _, ok := f.byID[j.ID]; !ok {
		return provisioning.ProvisioningJob{}, apperror.NotFound("provisioning job not found")
	}
	f.byID[j.ID] = j
	f.updateCalls = append(f.updateCalls, j)
	return j, nil
}

func (f *fakeProvisioningRepository) Delete(context.Context, uuid.UUID) error {
	panic("not used by these tests")
}

var _ provisioning.ProvisioningRepository = (*fakeProvisioningRepository)(nil)

type fakeServiceRepository struct {
	byID     map[uuid.UUID]domainservice.Service
	getCalls int
	err      error
}

func newFakeServiceRepository(services ...domainservice.Service) *fakeServiceRepository {
	f := &fakeServiceRepository{byID: make(map[uuid.UUID]domainservice.Service)}
	for _, s := range services {
		f.byID[s.ID] = s
	}
	return f
}

func (f *fakeServiceRepository) Get(_ context.Context, id uuid.UUID) (domainservice.Service, error) {
	f.getCalls++
	if f.err != nil {
		return domainservice.Service{}, f.err
	}
	s, ok := f.byID[id]
	if !ok {
		return domainservice.Service{}, apperror.NotFound("service not found")
	}
	return s, nil
}

func (f *fakeServiceRepository) List(context.Context) ([]domainservice.Service, error) {
	panic("not used by these tests")
}

func (f *fakeServiceRepository) Create(context.Context, domainservice.Service) (domainservice.Service, error) {
	panic("not used by these tests")
}

func (f *fakeServiceRepository) Update(context.Context, domainservice.Service) (domainservice.Service, error) {
	panic("not used by these tests")
}

func (f *fakeServiceRepository) Delete(context.Context, uuid.UUID) error {
	panic("not used by these tests")
}

var _ domainservice.ServiceRepository = (*fakeServiceRepository)(nil)

type fakeServiceEquipmentRepository struct {
	byServiceID map[uuid.UUID][]serviceequipment.ServiceEquipment
	listCalls   int
	err         error
}

func newFakeServiceEquipmentRepository(equipment ...serviceequipment.ServiceEquipment) *fakeServiceEquipmentRepository {
	f := &fakeServiceEquipmentRepository{byServiceID: make(map[uuid.UUID][]serviceequipment.ServiceEquipment)}
	for _, e := range equipment {
		f.byServiceID[e.ServiceID] = append(f.byServiceID[e.ServiceID], e)
	}
	return f
}

func (f *fakeServiceEquipmentRepository) Get(context.Context, uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	panic("not used by these tests")
}

func (f *fakeServiceEquipmentRepository) List(context.Context) ([]serviceequipment.ServiceEquipment, error) {
	panic("not used by these tests")
}

func (f *fakeServiceEquipmentRepository) Create(context.Context, serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	panic("not used by these tests")
}

func (f *fakeServiceEquipmentRepository) Update(context.Context, serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	panic("not used by these tests")
}

func (f *fakeServiceEquipmentRepository) Delete(context.Context, uuid.UUID) error {
	panic("not used by these tests")
}

func (f *fakeServiceEquipmentRepository) GetActiveByDeviceID(context.Context, uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	panic("not used by these tests")
}

func (f *fakeServiceEquipmentRepository) ListActiveByServiceID(_ context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error) {
	f.listCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.byServiceID[serviceID], nil
}

var _ serviceequipment.ServiceEquipmentRepository = (*fakeServiceEquipmentRepository)(nil)

// mockConnector records every call it receives (method name and the
// Request it was given) and can be configured to fail on specific
// methods — enough to prove both "the right method was called with the
// right data" and "a connector failure fails the job," goal 6's first
// and third requirements.
type mockConnector struct {
	name   string
	calls  []mockCall
	failOn map[string]error
}

type mockCall struct {
	method string
	req    connectors.Request
}

func newMockConnector(name string) *mockConnector {
	return &mockConnector{name: name, failOn: make(map[string]error)}
}

func (c *mockConnector) Name() string { return c.name }

func (c *mockConnector) invoke(method string, req connectors.Request) error {
	c.calls = append(c.calls, mockCall{method: method, req: req})
	return c.failOn[method]
}

func (c *mockConnector) Provision(_ context.Context, req connectors.Request) error {
	return c.invoke("Provision", req)
}
func (c *mockConnector) Reprovision(_ context.Context, req connectors.Request) error {
	return c.invoke("Reprovision", req)
}
func (c *mockConnector) Suspend(_ context.Context, req connectors.Request) error {
	return c.invoke("Suspend", req)
}
func (c *mockConnector) Resume(_ context.Context, req connectors.Request) error {
	return c.invoke("Resume", req)
}
func (c *mockConnector) Disconnect(_ context.Context, req connectors.Request) error {
	return c.invoke("Disconnect", req)
}
func (c *mockConnector) Synchronize(_ context.Context, req connectors.Request) error {
	return c.invoke("Synchronize", req)
}

var _ connectors.Connector = (*mockConnector)(nil)

var testNow = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func pendingJob(serviceID uuid.UUID, op provisioning.ProvisioningOperation) provisioning.ProvisioningJob {
	return provisioning.ProvisioningJob{
		ID:        uuid.New(),
		ServiceID: serviceID,
		Operation: op,
		Status:    provisioning.ProvisioningStatusPending,
	}
}

func testEquipment(serviceID uuid.UUID, role serviceequipment.EquipmentRole) serviceequipment.ServiceEquipment {
	return serviceequipment.ServiceEquipment{
		ID:        uuid.New(),
		ServiceID: serviceID,
		DeviceID:  uuid.New(),
		Role:      role,
	}
}

// -- tests -----------------------------------------------------------

// TestExecuteCallsCorrectConnectorMethodForEachOperation is goal 6's
// first explicit requirement: "correct connector methods are selected."
func TestExecuteCallsCorrectConnectorMethodForEachOperation(t *testing.T) {
	cases := []struct {
		operation provisioning.ProvisioningOperation
		method    string
	}{
		{provisioning.ProvisioningOperationProvision, "Provision"},
		{provisioning.ProvisioningOperationReprovision, "Reprovision"},
		{provisioning.ProvisioningOperationSuspend, "Suspend"},
		{provisioning.ProvisioningOperationResume, "Resume"},
		{provisioning.ProvisioningOperationDisconnect, "Disconnect"},
		{provisioning.ProvisioningOperationSynchronize, "Synchronize"},
	}

	for _, c := range cases {
		t.Run(string(c.operation), func(t *testing.T) {
			serviceID := uuid.New()
			svc := domainservice.Service{ID: serviceID}
			eq := testEquipment(serviceID, serviceequipment.EquipmentRoleONU)
			job := pendingJob(serviceID, c.operation)

			jobs := newFakeProvisioningRepository(job)
			services := newFakeServiceRepository(svc)
			equipment := newFakeServiceEquipmentRepository(eq)
			connector := newMockConnector("mock")
			registry := connectors.NewDefaultRegistry()
			registry.Register(serviceequipment.EquipmentRoleONU, connector)

			eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

			if err := eng.Execute(context.Background(), job.ID); err != nil {
				t.Fatalf("Execute() = %v", err)
			}

			if len(connector.calls) != 1 {
				t.Fatalf("len(calls) = %d, want 1; got %+v", len(connector.calls), connector.calls)
			}
			if connector.calls[0].method != c.method {
				t.Errorf("method called = %q, want %q", connector.calls[0].method, c.method)
			}
			if connector.calls[0].req.Equipment.ID != eq.ID {
				t.Errorf("Request.Equipment.ID = %v, want %v", connector.calls[0].req.Equipment.ID, eq.ID)
			}
			if connector.calls[0].req.Service.ID != serviceID {
				t.Errorf("Request.Service.ID = %v, want %v", connector.calls[0].req.Service.ID, serviceID)
			}
		})
	}
}

// TestExecuteTransitionsPendingToRunningToSucceeded is goal 6's "jobs
// transition correctly" and "successful execution transitions to
// Succeeded," verified by inspecting the exact sequence of persisted
// states, not just the final one.
func TestExecuteTransitionsPendingToRunningToSucceeded(t *testing.T) {
	serviceID := uuid.New()
	svc := domainservice.Service{ID: serviceID}
	eq := testEquipment(serviceID, serviceequipment.EquipmentRoleONU)
	job := pendingJob(serviceID, provisioning.ProvisioningOperationProvision)

	jobs := newFakeProvisioningRepository(job)
	services := newFakeServiceRepository(svc)
	equipment := newFakeServiceEquipmentRepository(eq)
	registry := connectors.NewDefaultRegistry()
	registry.Register(serviceequipment.EquipmentRoleONU, newMockConnector("mock"))

	eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

	if err := eng.Execute(context.Background(), job.ID); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(jobs.updateCalls) != 2 {
		t.Fatalf("len(updateCalls) = %d, want 2 (Running, then Succeeded); got %+v", len(jobs.updateCalls), jobs.updateCalls)
	}
	if jobs.updateCalls[0].Status != provisioning.ProvisioningStatusRunning {
		t.Errorf("first persisted Status = %q, want %q", jobs.updateCalls[0].Status, provisioning.ProvisioningStatusRunning)
	}
	if jobs.updateCalls[0].StartedAt == nil || !jobs.updateCalls[0].StartedAt.Equal(testNow) {
		t.Errorf("first persisted StartedAt = %v, want %v", jobs.updateCalls[0].StartedAt, testNow)
	}
	if jobs.updateCalls[1].Status != provisioning.ProvisioningStatusSucceeded {
		t.Errorf("second persisted Status = %q, want %q", jobs.updateCalls[1].Status, provisioning.ProvisioningStatusSucceeded)
	}
	if jobs.updateCalls[1].CompletedAt == nil || !jobs.updateCalls[1].CompletedAt.Equal(testNow) {
		t.Errorf("second persisted CompletedAt = %v, want %v", jobs.updateCalls[1].CompletedAt, testNow)
	}

	final := jobs.byID[job.ID]
	if final.Status != provisioning.ProvisioningStatusSucceeded {
		t.Errorf("final Status = %q, want %q", final.Status, provisioning.ProvisioningStatusSucceeded)
	}
}

// TestExecuteTransitionsToFailedOnConnectorError is goal 6's "connector
// failures transition jobs to Failed."
func TestExecuteTransitionsToFailedOnConnectorError(t *testing.T) {
	serviceID := uuid.New()
	svc := domainservice.Service{ID: serviceID}
	eq := testEquipment(serviceID, serviceequipment.EquipmentRoleONU)
	job := pendingJob(serviceID, provisioning.ProvisioningOperationProvision)

	jobs := newFakeProvisioningRepository(job)
	services := newFakeServiceRepository(svc)
	equipment := newFakeServiceEquipmentRepository(eq)
	connector := newMockConnector("mock")
	connectorErr := errors.New("device unreachable")
	connector.failOn["Provision"] = connectorErr
	registry := connectors.NewDefaultRegistry()
	registry.Register(serviceequipment.EquipmentRoleONU, connector)

	eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

	err := eng.Execute(context.Background(), job.ID)
	if err == nil {
		t.Fatal("Execute() = nil, want an error")
	}
	if !errors.Is(err, connectorErr) {
		t.Errorf("Execute() error = %v, want it to wrap %v", err, connectorErr)
	}

	final := jobs.byID[job.ID]
	if final.Status != provisioning.ProvisioningStatusFailed {
		t.Errorf("final Status = %q, want %q", final.Status, provisioning.ProvisioningStatusFailed)
	}
	if final.ErrorMessage == nil {
		t.Fatal("ErrorMessage = nil, want the connector failure recorded")
	}
	if final.CompletedAt == nil || !final.CompletedAt.Equal(testNow) {
		t.Errorf("CompletedAt = %v, want %v", final.CompletedAt, testNow)
	}
}

// TestExecuteFailsFastAfterFirstConnectorFailure proves a second piece of
// equipment's connector is never called once an earlier one has already
// failed the job — consistent with "no retries, no queues" (there is
// nothing to gain from continuing).
func TestExecuteFailsFastAfterFirstConnectorFailure(t *testing.T) {
	serviceID := uuid.New()
	svc := domainservice.Service{ID: serviceID}
	failingEquipment := testEquipment(serviceID, serviceequipment.EquipmentRoleONU)
	otherEquipment := testEquipment(serviceID, serviceequipment.EquipmentRoleRouter)
	job := pendingJob(serviceID, provisioning.ProvisioningOperationProvision)

	jobs := newFakeProvisioningRepository(job)
	services := newFakeServiceRepository(svc)
	equipment := newFakeServiceEquipmentRepository(failingEquipment, otherEquipment)

	failingConnector := newMockConnector("failing")
	failingConnector.failOn["Provision"] = errors.New("device unreachable")
	otherConnector := newMockConnector("other")

	registry := connectors.NewDefaultRegistry()
	registry.Register(serviceequipment.EquipmentRoleONU, failingConnector)
	registry.Register(serviceequipment.EquipmentRoleRouter, otherConnector)

	eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

	if err := eng.Execute(context.Background(), job.ID); err == nil {
		t.Fatal("Execute() = nil, want an error")
	}

	if len(otherConnector.calls) != 0 {
		t.Errorf("otherConnector.calls = %+v, want none — execution must stop at the first failure", otherConnector.calls)
	}
}

// TestExecuteDoesNotCallConnectorWhenJobNotPending is goal 6's "no
// connector is called when the job is not Pending," checked for every
// non-Pending status.
func TestExecuteDoesNotCallConnectorWhenJobNotPending(t *testing.T) {
	for _, status := range []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusRunning,
		provisioning.ProvisioningStatusSucceeded,
		provisioning.ProvisioningStatusFailed,
		provisioning.ProvisioningStatusCancelled,
	} {
		t.Run(string(status), func(t *testing.T) {
			serviceID := uuid.New()
			job := pendingJob(serviceID, provisioning.ProvisioningOperationProvision)
			job.Status = status

			jobs := newFakeProvisioningRepository(job)
			services := newFakeServiceRepository(domainservice.Service{ID: serviceID})
			equipment := newFakeServiceEquipmentRepository(testEquipment(serviceID, serviceequipment.EquipmentRoleONU))
			connector := newMockConnector("mock")
			registry := connectors.NewDefaultRegistry()
			registry.Register(serviceequipment.EquipmentRoleONU, connector)

			eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

			err := eng.Execute(context.Background(), job.ID)

			if !apperror.Is(err, apperror.KindConflict) {
				t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
			}
			if len(connector.calls) != 0 {
				t.Errorf("connector.calls = %+v, want none", connector.calls)
			}
			if services.getCalls != 0 {
				t.Errorf("services.getCalls = %d, want 0 — the Service must not be loaded either", services.getCalls)
			}
			if equipment.listCalls != 0 {
				t.Errorf("equipment.listCalls = %d, want 0 — equipment must not be loaded either", equipment.listCalls)
			}
			if len(jobs.updateCalls) != 0 {
				t.Errorf("updateCalls = %+v, want none — the job must not be written at all", jobs.updateCalls)
			}
		})
	}
}

func TestExecutePropagatesNotFoundWhenJobDoesNotExist(t *testing.T) {
	jobs := newFakeProvisioningRepository()
	services := newFakeServiceRepository()
	equipment := newFakeServiceEquipmentRepository()
	registry := connectors.NewDefaultRegistry()

	eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

	err := eng.Execute(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

// TestExecuteFailsJobWhenServiceLoadFails proves a failure loading the
// Service itself (not a connector) still fails the job through the same
// path, after the job has already been moved to Running.
func TestExecuteFailsJobWhenServiceLoadFails(t *testing.T) {
	serviceID := uuid.New()
	job := pendingJob(serviceID, provisioning.ProvisioningOperationProvision)

	jobs := newFakeProvisioningRepository(job)
	services := newFakeServiceRepository()
	services.err = apperror.NotFound("service not found")
	equipment := newFakeServiceEquipmentRepository()
	registry := connectors.NewDefaultRegistry()

	eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

	err := eng.Execute(context.Background(), job.ID)
	if err == nil {
		t.Fatal("Execute() = nil, want an error")
	}

	final := jobs.byID[job.ID]
	if final.Status != provisioning.ProvisioningStatusFailed {
		t.Errorf("final Status = %q, want %q", final.Status, provisioning.ProvisioningStatusFailed)
	}
	if len(jobs.updateCalls) != 2 {
		t.Fatalf("len(updateCalls) = %d, want 2 (Running, then Failed)", len(jobs.updateCalls))
	}
}

// TestExecuteFailsJobWhenNoConnectorRegisteredForRole proves a
// configuration gap (equipment whose Role has no registered connector)
// fails the job loudly rather than being silently skipped.
func TestExecuteFailsJobWhenNoConnectorRegisteredForRole(t *testing.T) {
	serviceID := uuid.New()
	svc := domainservice.Service{ID: serviceID}
	eq := testEquipment(serviceID, serviceequipment.EquipmentRoleUPS)
	job := pendingJob(serviceID, provisioning.ProvisioningOperationProvision)

	jobs := newFakeProvisioningRepository(job)
	services := newFakeServiceRepository(svc)
	equipment := newFakeServiceEquipmentRepository(eq)
	registry := connectors.NewDefaultRegistry() // nothing registered for UPS

	eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

	err := eng.Execute(context.Background(), job.ID)
	if err == nil {
		t.Fatal("Execute() = nil, want an error")
	}

	final := jobs.byID[job.ID]
	if final.Status != provisioning.ProvisioningStatusFailed {
		t.Errorf("final Status = %q, want %q", final.Status, provisioning.ProvisioningStatusFailed)
	}
}

// TestExecuteSucceedsWithNoActiveEquipment proves a Service with no
// active Service Equipment is not itself a failure: Execute calls no
// connector and still reaches Succeeded.
func TestExecuteSucceedsWithNoActiveEquipment(t *testing.T) {
	serviceID := uuid.New()
	svc := domainservice.Service{ID: serviceID}
	job := pendingJob(serviceID, provisioning.ProvisioningOperationProvision)

	jobs := newFakeProvisioningRepository(job)
	services := newFakeServiceRepository(svc)
	equipment := newFakeServiceEquipmentRepository() // none
	registry := connectors.NewDefaultRegistry()

	eng := engine.NewDefaultEngine(jobs, services, equipment, registry, clock.NewFrozen(testNow))

	if err := eng.Execute(context.Background(), job.ID); err != nil {
		t.Fatalf("Execute() = %v, want success for a service with no active equipment", err)
	}

	final := jobs.byID[job.ID]
	if final.Status != provisioning.ProvisioningStatusSucceeded {
		t.Errorf("final Status = %q, want %q", final.Status, provisioning.ProvisioningStatusSucceeded)
	}
}
