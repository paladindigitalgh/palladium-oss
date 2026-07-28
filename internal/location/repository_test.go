package location_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
)

// stubLocationRepository has no SQL implementation to test yet — that is
// internal/location/postgres's job. It exists solely to prove
// LocationRepository is satisfiable with a sane, consistent method shape,
// mirroring internal/customer/repository_test.go's stub for the same
// reason: the var block's compile-time assertion is the actual check —
// this file fails to build if the interface and this stub ever drift
// apart.
type stubLocationRepository struct{}

func (stubLocationRepository) Get(context.Context, uuid.UUID) (location.Location, error) {
	return location.Location{}, nil
}
func (stubLocationRepository) List(context.Context) ([]location.Location, error) { return nil, nil }
func (stubLocationRepository) Create(_ context.Context, l location.Location) (location.Location, error) {
	return l, nil
}
func (stubLocationRepository) Update(_ context.Context, l location.Location) (location.Location, error) {
	return l, nil
}
func (stubLocationRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ location.LocationRepository = (*stubLocationRepository)(nil)

func TestLocationRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, LocationRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
