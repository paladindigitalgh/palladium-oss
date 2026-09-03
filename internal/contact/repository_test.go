package contact_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
)

// stubContactRepository has no SQL implementation to test yet — that is
// internal/contact/postgres's job. It exists solely to prove
// ContactRepository is satisfiable with a sane, consistent method shape,
// mirroring internal/location/repository_test.go's stub for the same
// reason: the var block's compile-time assertion is the actual check —
// this file fails to build if the interface and this stub ever drift
// apart.
type stubContactRepository struct{}

func (stubContactRepository) Get(context.Context, uuid.UUID) (contact.Contact, error) {
	return contact.Contact{}, nil
}
func (stubContactRepository) List(context.Context) ([]contact.Contact, error) { return nil, nil }
func (stubContactRepository) Create(_ context.Context, c contact.Contact) (contact.Contact, error) {
	return c, nil
}
func (stubContactRepository) Update(_ context.Context, c contact.Contact) (contact.Contact, error) {
	return c, nil
}
func (stubContactRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ contact.ContactRepository = (*stubContactRepository)(nil)

func TestContactRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ContactRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
