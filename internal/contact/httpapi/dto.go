// Package httpapi is the Contact domain's REST layer. It depends on
// internal/contact/service, never on a repository directly, and never
// exposes internal/contact's domain types over the wire — see the DTOs
// in this file. It mirrors internal/location/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
)

// contactRequest is the JSON body for POST /api/v1/contacts and
// PUT /api/v1/contacts/{id}.
//
// Role and Status are plain strings here, not contact.ContactRole /
// contact.ContactStatus, even though those types would marshal to the
// same JSON today — the same "DTOs only" separation
// internal/location/httpapi.locationRequest documents. The conversion
// happens once, explicitly, in toContact below; ContactService.Create/
// Update reject an unrecognized value via Contact.Validate exactly as
// they would for a request built any other way.
//
// CustomerID is left as its plain primitive type (uuid.UUID) for the
// same reason internal/location/httpapi.locationRequest's CustomerID is:
// it carries no domain enum type to decouple from in the first place.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type contactRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Status     string    `json:"status"`

	Description string `json:"description"`
}

// toContact converts a request into a domain contact.Contact. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req contactRequest) toContact(id uuid.UUID) contact.Contact {
	return contact.Contact{
		ID:         id,
		CustomerID: req.CustomerID,
		Name:       req.Name,
		Role:       contact.ContactRole(req.Role),
		Email:      req.Email,
		Phone:      req.Phone,
		Status:     contact.ContactStatus(req.Status),

		Description: req.Description,
	}
}

// contactResponse is the JSON representation of a Contact returned to
// clients. Decoupling the wire format from contact.Contact's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type contactResponse struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Status     string    `json:"status"`

	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newContactResponse(c contact.Contact) contactResponse {
	return contactResponse{
		ID:         c.ID,
		CustomerID: c.CustomerID,
		Name:       c.Name,
		Role:       string(c.Role),
		Email:      c.Email,
		Phone:      c.Phone,
		Status:     string(c.Status),

		Description: c.Description,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// contactListResponse wraps a slice of contacts in an object rather than
// returning a bare JSON array — the same reasoning as
// internal/location/httpapi's locationListResponse.
type contactListResponse struct {
	Contacts []contactResponse `json:"contacts"`
}

func newContactListResponse(contacts []contact.Contact) contactListResponse {
	resp := contactListResponse{Contacts: make([]contactResponse, len(contacts))}
	for i, c := range contacts {
		resp.Contacts[i] = newContactResponse(c)
	}
	return resp
}
