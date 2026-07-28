package accessinterface_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
)

// stubAccessInterfaceRepository has no SQL implementation to test yet —
// that is internal/accessinterface/postgres's job. It exists solely to
// prove AccessInterfaceRepository is satisfiable with a sane, consistent
// method shape, mirroring internal/olt/repository_test.go's stub for the
// same reason: the var block's compile-time assertion is the actual
// check — this file fails to build if the interface and this stub ever
// drift apart.
type stubAccessInterfaceRepository struct{}

func (stubAccessInterfaceRepository) Get(context.Context, uuid.UUID) (accessinterface.AccessInterface, error) {
	return accessinterface.AccessInterface{}, nil
}
func (stubAccessInterfaceRepository) List(context.Context) ([]accessinterface.AccessInterface, error) {
	return nil, nil
}
func (stubAccessInterfaceRepository) Create(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	return a, nil
}
func (stubAccessInterfaceRepository) Update(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	return a, nil
}
func (stubAccessInterfaceRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ accessinterface.AccessInterfaceRepository = (*stubAccessInterfaceRepository)(nil)

func TestAccessInterfaceRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, AccessInterfaceRepository has the intended
	// Get/List/Create/Update/Delete shape. This test exists so `go test`
	// reports that check explicitly instead of the file silently
	// containing no tests.
}
