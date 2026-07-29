package authentication_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
)

// stubAuthenticationRepository has no SQL implementation to test yet —
// that is internal/authentication/postgres's job. It exists solely to
// prove AuthenticationRepository is satisfiable with a sane, consistent
// method shape, mirroring internal/catalog/repository_test.go's stub for
// the same reason: the var block's compile-time assertion is the actual
// check — this file fails to build if the interface and this stub ever
// drift apart.
type stubAuthenticationRepository struct{}

func (stubAuthenticationRepository) Get(context.Context, uuid.UUID) (authentication.Authentication, error) {
	return authentication.Authentication{}, nil
}
func (stubAuthenticationRepository) List(context.Context) ([]authentication.Authentication, error) {
	return nil, nil
}
func (stubAuthenticationRepository) Create(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	return a, nil
}
func (stubAuthenticationRepository) Update(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	return a, nil
}
func (stubAuthenticationRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ authentication.AuthenticationRepository = (*stubAuthenticationRepository)(nil)

func TestAuthenticationRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, AuthenticationRepository has the intended
	// Get/List/Create/Update/Delete shape. This test exists so `go test`
	// reports that check explicitly instead of the file silently
	// containing no tests.
}
