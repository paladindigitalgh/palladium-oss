package ponport_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// stubPONPortRepository has no SQL implementation to test yet — that is
// internal/ponport/postgres's job. It exists solely to prove
// PONPortRepository is satisfiable with a sane, consistent method shape,
// mirroring internal/olt/repository_test.go's stub for the same reason:
// the var block's compile-time assertion is the actual check — this
// file fails to build if the interface and this stub ever drift apart.
type stubPONPortRepository struct{}

func (stubPONPortRepository) Get(context.Context, uuid.UUID) (ponport.PONPort, error) {
	return ponport.PONPort{}, nil
}
func (stubPONPortRepository) List(context.Context) ([]ponport.PONPort, error) { return nil, nil }
func (stubPONPortRepository) Create(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	return p, nil
}
func (stubPONPortRepository) Update(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	return p, nil
}
func (stubPONPortRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ ponport.PONPortRepository = (*stubPONPortRepository)(nil)

func TestPONPortRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, PONPortRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
