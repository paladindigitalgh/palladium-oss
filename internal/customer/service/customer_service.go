// Package service is the Customer domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/customer/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/customer/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/inventory/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
)

// CustomerService is the Customer domain's business logic.
//
// It depends only on customer.CustomerRepository — not clock.Clock, for
// the same reason internal/inventory/service.SiteService does not:
// timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now". Adding
// an unused clock.Clock parameter here just because other constructors
// have one would be exactly the unnecessary dependency CLAUDE.md warns
// against.
type CustomerService struct {
	customers customer.CustomerRepository
}

// NewCustomerService builds a CustomerService.
func NewCustomerService(customers customer.CustomerRepository) *CustomerService {
	return &CustomerService{customers: customers}
}

// Get retrieves a Customer by ID.
func (s *CustomerService) Get(ctx context.Context, id uuid.UUID) (customer.Customer, error) {
	return s.customers.Get(ctx, id)
}

// List returns every Customer.
func (s *CustomerService) List(ctx context.Context) ([]customer.Customer, error) {
	return s.customers.List(ctx)
}

// Create validates c and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of CustomerService — so every caller
// gets the same guarantee for free, and invalid input never costs a
// database round trip. See internal/inventory/service.SiteService.Create
// for the identical reasoning applied to Sites.
func (s *CustomerService) Create(ctx context.Context, c customer.Customer) (customer.Customer, error) {
	if err := c.Validate(); err != nil {
		return customer.Customer{}, err
	}
	return s.customers.Create(ctx, c)
}

// Update validates c and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *CustomerService) Update(ctx context.Context, c customer.Customer) (customer.Customer, error) {
	if err := c.Validate(); err != nil {
		return customer.Customer{}, err
	}
	return s.customers.Update(ctx, c)
}

// Delete removes the Customer identified by id.
func (s *CustomerService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.customers.Delete(ctx, id)
}
