// Package httpapi is the Customer domain's REST layer. It depends on
// internal/customer/service, never on a repository directly, and never
// exposes internal/customer's domain types over the wire — see the DTOs
// in this file. It mirrors internal/inventory/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
)

// customerRequest is the JSON body for POST /api/v1/customers and
// PUT /api/v1/customers/{id}.
//
// CustomerType and Status are plain strings here, not customer.CustomerType
// / customer.CustomerStatus, even though those types would marshal to the
// same JSON today (they are just named string types with no custom
// encoding). Using the domain's enum types directly in a DTO would still
// couple the wire format to internal/customer's Go types instead of only
// to the JSON shape they happen to produce right now — the same "DTOs
// only" separation this milestone asks for, applied to individual fields,
// not just whole structs. The conversion happens once, explicitly, in
// toCustomer below; CustomerService.Create/Update reject an unrecognized
// value via Customer.Validate (see internal/customer/validate.go) exactly
// as they would for a request built any other way — this handler does not
// duplicate that check (goal 4: validation is the service's job).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set (see
// CustomerRepository.Create's doc comment in
// internal/customer/postgres/customer.go for the same rule one layer
// down).
type customerRequest struct {
	Name         string `json:"name"`
	CustomerType string `json:"customer_type"`
	Status       string `json:"status"`
	Description  string `json:"description"`
}

// toCustomer converts a request into a domain customer.Customer. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req customerRequest) toCustomer(id uuid.UUID) customer.Customer {
	return customer.Customer{
		ID:           id,
		Name:         req.Name,
		CustomerType: customer.CustomerType(req.CustomerType),
		Status:       customer.CustomerStatus(req.Status),
		Description:  req.Description,
	}
}

// customerResponse is the JSON representation of a Customer returned to
// clients. Decoupling the wire format from customer.Customer's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type customerResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	CustomerType string    `json:"customer_type"`
	Status       string    `json:"status"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func newCustomerResponse(c customer.Customer) customerResponse {
	return customerResponse{
		ID:           c.ID,
		Name:         c.Name,
		CustomerType: string(c.CustomerType),
		Status:       string(c.Status),
		Description:  c.Description,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// customerListResponse wraps a slice of customers in an object rather
// than returning a bare JSON array — the same reasoning as
// internal/inventory/httpapi's siteListResponse: a bare top-level array
// can never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding a
// field next to "customers" is not.
type customerListResponse struct {
	Customers []customerResponse `json:"customers"`
}

func newCustomerListResponse(customers []customer.Customer) customerListResponse {
	resp := customerListResponse{Customers: make([]customerResponse, len(customers))}
	for i, c := range customers {
		resp.Customers[i] = newCustomerResponse(c)
	}
	return resp
}
