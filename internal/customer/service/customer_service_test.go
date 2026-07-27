package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/customer/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeCustomerRepository is an in-memory customer.CustomerRepository. Like
// internal/inventory/service/site_service_test.go's fakeSiteRepository, it
// exists so CustomerService's business logic — validate, then delegate —
// is tested without a real database;
// internal/customer/postgres/customer_test.go already covers the
// repository itself against real PostgreSQL. It tracks whether
// Create/Update were actually invoked, which is what lets
// TestCustomerServiceCreateRejectsInvalidCustomerWithoutPersisting prove
// validation happens before any repository call.
type fakeCustomerRepository struct {
	byID         map[uuid.UUID]customer.Customer
	createCalled bool
	updateCalled bool
}

func newFakeCustomerRepository(customers ...customer.Customer) *fakeCustomerRepository {
	f := &fakeCustomerRepository{byID: make(map[uuid.UUID]customer.Customer)}
	for _, c := range customers {
		f.byID[c.ID] = c
	}
	return f
}

func (f *fakeCustomerRepository) Get(_ context.Context, id uuid.UUID) (customer.Customer, error) {
	c, ok := f.byID[id]
	if !ok {
		return customer.Customer{}, apperror.NotFound("customer not found")
	}
	return c, nil
}

func (f *fakeCustomerRepository) List(_ context.Context) ([]customer.Customer, error) {
	customers := make([]customer.Customer, 0, len(f.byID))
	for _, c := range f.byID {
		customers = append(customers, c)
	}
	return customers, nil
}

func (f *fakeCustomerRepository) Create(_ context.Context, c customer.Customer) (customer.Customer, error) {
	f.createCalled = true
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeCustomerRepository) Update(_ context.Context, c customer.Customer) (customer.Customer, error) {
	f.updateCalled = true
	if _, ok := f.byID[c.ID]; !ok {
		return customer.Customer{}, apperror.NotFound("customer not found")
	}
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeCustomerRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("customer not found")
	}
	delete(f.byID, id)
	return nil
}

var _ customer.CustomerRepository = (*fakeCustomerRepository)(nil)

func validCustomer() customer.Customer {
	return customer.Customer{
		Name:         "Jane Doe",
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
	}
}

func TestCustomerServiceCreateSucceeds(t *testing.T) {
	repo := newFakeCustomerRepository()
	svc := service.NewCustomerService(repo)

	created, err := svc.Create(context.Background(), validCustomer())
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

func TestCustomerServiceCreateRejectsInvalidCustomerWithoutPersisting(t *testing.T) {
	repo := newFakeCustomerRepository()
	svc := service.NewCustomerService(repo)

	_, err := svc.Create(context.Background(), customer.Customer{}) // no Name, CustomerType, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestCustomerServiceUpdateSucceeds(t *testing.T) {
	existing := validCustomer()
	existing.ID = uuid.New()
	repo := newFakeCustomerRepository(existing)
	svc := service.NewCustomerService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = customer.CustomerStatusInactive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != customer.CustomerStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, customer.CustomerStatusInactive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestCustomerServiceUpdateRejectsInvalidCustomerWithoutPersisting(t *testing.T) {
	existing := validCustomer()
	existing.ID = uuid.New()
	repo := newFakeCustomerRepository(existing)
	svc := service.NewCustomerService(repo)

	invalid := existing
	invalid.Name = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestCustomerServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeCustomerRepository()
	svc := service.NewCustomerService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestCustomerServiceListDelegatesToRepository(t *testing.T) {
	a := validCustomer()
	a.ID = uuid.New()
	b := validCustomer()
	b.ID = uuid.New()
	repo := newFakeCustomerRepository(a, b)
	svc := service.NewCustomerService(repo)

	customers, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(customers) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(customers))
	}
}

func TestCustomerServiceDeleteSucceeds(t *testing.T) {
	existing := validCustomer()
	existing.ID = uuid.New()
	repo := newFakeCustomerRepository(existing)
	svc := service.NewCustomerService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestCustomerServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeCustomerRepository()
	svc := service.NewCustomerService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
