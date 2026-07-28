package olt_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
)

// stubOLTRepository has no SQL implementation to test yet — that is
// internal/olt/postgres's job. It exists solely to prove OLTRepository
// is satisfiable with a sane, consistent method shape, mirroring
// internal/product/repository_test.go's stub for the same reason: the
// var block's compile-time assertion is the actual check — this file
// fails to build if the interface and this stub ever drift apart.
type stubOLTRepository struct{}

func (stubOLTRepository) Get(context.Context, uuid.UUID) (olt.OLT, error) {
	return olt.OLT{}, nil
}
func (stubOLTRepository) List(context.Context) ([]olt.OLT, error) { return nil, nil }
func (stubOLTRepository) Create(_ context.Context, o olt.OLT) (olt.OLT, error) {
	return o, nil
}
func (stubOLTRepository) Update(_ context.Context, o olt.OLT) (olt.OLT, error) {
	return o, nil
}
func (stubOLTRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ olt.OLTRepository = (*stubOLTRepository)(nil)

func TestOLTRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, OLTRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
