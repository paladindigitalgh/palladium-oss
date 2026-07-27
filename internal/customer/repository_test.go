package customer_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
)

// stubCustomerRepository has no SQL implementation to test yet — that is
// internal/customer/postgres's job. It exists solely to prove
// CustomerRepository is satisfiable with a sane, consistent method shape,
// mirroring internal/inventory/repository_test.go's stubs for the same
// reason: the var block's compile-time assertion is the actual check —
// this file fails to build if the interface and this stub ever drift
// apart.
type stubCustomerRepository struct{}

func (stubCustomerRepository) Get(context.Context, uuid.UUID) (customer.Customer, error) {
	return customer.Customer{}, nil
}
func (stubCustomerRepository) List(context.Context) ([]customer.Customer, error) { return nil, nil }
func (stubCustomerRepository) Create(_ context.Context, c customer.Customer) (customer.Customer, error) {
	return c, nil
}
func (stubCustomerRepository) Update(_ context.Context, c customer.Customer) (customer.Customer, error) {
	return c, nil
}
func (stubCustomerRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ customer.CustomerRepository = (*stubCustomerRepository)(nil)

func TestCustomerRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, CustomerRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
