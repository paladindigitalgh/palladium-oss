package serviceequipment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// stubServiceEquipmentRepository has no SQL implementation to test yet —
// that is internal/serviceequipment/postgres's job. It exists solely to
// prove ServiceEquipmentRepository is satisfiable with a sane, consistent
// method shape, mirroring internal/service/repository_test.go's stub for
// the same reason: the var block's compile-time assertion is the actual
// check — this file fails to build if the interface and this stub ever
// drift apart. GetActiveByDeviceID always reports "not found," matching
// what the real repository returns for a Device with no active
// assignment.
type stubServiceEquipmentRepository struct{}

func (stubServiceEquipmentRepository) Get(context.Context, uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	return serviceequipment.ServiceEquipment{}, nil
}
func (stubServiceEquipmentRepository) List(context.Context) ([]serviceequipment.ServiceEquipment, error) {
	return nil, nil
}
func (stubServiceEquipmentRepository) Create(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	return e, nil
}
func (stubServiceEquipmentRepository) Update(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	return e, nil
}
func (stubServiceEquipmentRepository) Delete(context.Context, uuid.UUID) error { return nil }
func (stubServiceEquipmentRepository) GetActiveByDeviceID(context.Context, uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	return serviceequipment.ServiceEquipment{}, apperror.NotFound("no active assignment")
}
func (stubServiceEquipmentRepository) ListActiveByServiceID(context.Context, uuid.UUID) ([]serviceequipment.ServiceEquipment, error) {
	return nil, nil
}

var _ serviceequipment.ServiceEquipmentRepository = (*stubServiceEquipmentRepository)(nil)

func TestServiceEquipmentRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ServiceEquipmentRepository has the intended
	// Get/List/Create/Update/Delete/GetActiveByDeviceID/
	// ListActiveByServiceID shape. This test exists so `go test` reports
	// that check explicitly instead of the file silently containing no
	// tests.
}
