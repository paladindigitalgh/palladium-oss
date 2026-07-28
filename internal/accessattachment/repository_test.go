package accessattachment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// stubAccessAttachmentRepository has no SQL implementation to test yet —
// that is internal/accessattachment/postgres's job. It exists solely to
// prove AccessAttachmentRepository is satisfiable with a sane, consistent
// method shape, mirroring
// internal/serviceequipment/repository_test.go's stub for the same
// reason: the var block's compile-time assertion is the actual check —
// this file fails to build if the interface and this stub ever drift
// apart. GetActiveByServiceEquipmentID always reports "not found,"
// matching what the real repository returns for equipment with no active
// attachment.
type stubAccessAttachmentRepository struct{}

func (stubAccessAttachmentRepository) Get(context.Context, uuid.UUID) (accessattachment.AccessAttachment, error) {
	return accessattachment.AccessAttachment{}, nil
}
func (stubAccessAttachmentRepository) List(context.Context) ([]accessattachment.AccessAttachment, error) {
	return nil, nil
}
func (stubAccessAttachmentRepository) Create(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	return a, nil
}
func (stubAccessAttachmentRepository) Update(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	return a, nil
}
func (stubAccessAttachmentRepository) Delete(context.Context, uuid.UUID) error { return nil }
func (stubAccessAttachmentRepository) GetActiveByServiceEquipmentID(context.Context, uuid.UUID) (accessattachment.AccessAttachment, error) {
	return accessattachment.AccessAttachment{}, apperror.NotFound("no active attachment")
}

var _ accessattachment.AccessAttachmentRepository = (*stubAccessAttachmentRepository)(nil)

func TestAccessAttachmentRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, AccessAttachmentRepository has the intended
	// Get/List/Create/Update/Delete/GetActiveByServiceEquipmentID shape.
	// This test exists so `go test` reports that check explicitly instead
	// of the file silently containing no tests.
}
