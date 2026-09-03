// Package service is the Contact domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/contact/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/contact/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/location/service exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
)

// ContactService is the Contact domain's business logic.
//
// It depends only on contact.ContactRepository — not clock.Clock, for
// the same reason internal/location/service.LocationService does not:
// timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now".
type ContactService struct {
	contacts contact.ContactRepository
}

// NewContactService builds a ContactService.
func NewContactService(contacts contact.ContactRepository) *ContactService {
	return &ContactService{contacts: contacts}
}

// Get retrieves a Contact by ID.
func (s *ContactService) Get(ctx context.Context, id uuid.UUID) (contact.Contact, error) {
	return s.contacts.Get(ctx, id)
}

// List returns every Contact.
func (s *ContactService) List(ctx context.Context) ([]contact.Contact, error) {
	return s.contacts.List(ctx)
}

// Create validates c and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of ContactService — so every caller
// gets the same guarantee for free, and invalid input never costs a
// database round trip. See internal/location/service.LocationService.Create
// for the identical reasoning applied to Locations.
func (s *ContactService) Create(ctx context.Context, c contact.Contact) (contact.Contact, error) {
	if err := c.Validate(); err != nil {
		return contact.Contact{}, err
	}
	return s.contacts.Create(ctx, c)
}

// Update validates c and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *ContactService) Update(ctx context.Context, c contact.Contact) (contact.Contact, error) {
	if err := c.Validate(); err != nil {
		return contact.Contact{}, err
	}
	return s.contacts.Update(ctx, c)
}

// Delete removes the Contact identified by id.
func (s *ContactService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.contacts.Delete(ctx, id)
}
