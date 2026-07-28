package accessnetwork_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
)

// stubAccessNetworkRepository has no SQL implementation to test yet —
// that is internal/accessnetwork/postgres's job. It exists solely to
// prove AccessNetworkRepository is satisfiable with a sane, consistent
// method shape, mirroring internal/catalog/repository_test.go's stub for
// the same reason: the var block's compile-time assertion is the actual
// check — this file fails to build if the interface and this stub ever
// drift apart.
type stubAccessNetworkRepository struct{}

func (stubAccessNetworkRepository) Get(context.Context, uuid.UUID) (accessnetwork.AccessNetwork, error) {
	return accessnetwork.AccessNetwork{}, nil
}
func (stubAccessNetworkRepository) List(context.Context) ([]accessnetwork.AccessNetwork, error) {
	return nil, nil
}
func (stubAccessNetworkRepository) Create(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	return a, nil
}
func (stubAccessNetworkRepository) Update(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	return a, nil
}
func (stubAccessNetworkRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ accessnetwork.AccessNetworkRepository = (*stubAccessNetworkRepository)(nil)

func TestAccessNetworkRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, AccessNetworkRepository has the intended
	// Get/List/Create/Update/Delete shape. This test exists so `go test`
	// reports that check explicitly instead of the file silently
	// containing no tests.
}
