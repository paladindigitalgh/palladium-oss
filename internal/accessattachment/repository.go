package accessattachment

import (
	"context"

	"github.com/google/uuid"
)

// AccessAttachmentRepository persists AccessAttachment records. Get,
// List, Create, Update, and Delete follow the exact shape of every other
// repository in this codebase (see e.g.
// serviceequipment.ServiceEquipmentRepository): Create and Update return
// the persisted entity so a caller sees anything the store sets (e.g.
// timestamps) without a second read.
//
// GetActiveByServiceEquipmentID is this package's one addition beyond
// the standard shape, mirroring
// serviceequipment.ServiceEquipmentRepository.GetActiveByDeviceID
// exactly: it exists specifically to support the
// active-attachment-uniqueness business rule AccessAttachmentService
// enforces (see internal/accessattachment/service) — before creating a
// new attachment, the service layer asks "does this ServiceEquipment
// already have an active attachment," and this is the query that
// answers it directly, in one round trip, rather than every caller
// fetching List and filtering client-side. Like Get, it returns an
// apperror.KindNotFound error when no active attachment exists for
// serviceEquipmentID — that is the expected, common case (equipment with
// no current attachment), not an exceptional one.
//
// Nothing in this package implements AccessAttachmentRepository — no
// SQL, no migrations — so the domain has zero dependency on any storage
// technology. A concrete implementation
// (internal/accessattachment/postgres) satisfies it.
type AccessAttachmentRepository interface {
	Get(ctx context.Context, id uuid.UUID) (AccessAttachment, error)
	List(ctx context.Context) ([]AccessAttachment, error)
	Create(ctx context.Context, attachment AccessAttachment) (AccessAttachment, error)
	Update(ctx context.Context, attachment AccessAttachment) (AccessAttachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveByServiceEquipmentID(ctx context.Context, serviceEquipmentID uuid.UUID) (AccessAttachment, error)
}
