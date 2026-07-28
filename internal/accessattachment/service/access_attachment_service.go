// Package service is the Access Attachment domain's business logic
// layer. It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/accessattachment/httpapi), and repositories never validate or
// otherwise reason about business rules (see
// internal/accessattachment/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/serviceequipment/service exactly, including that package's
// active-assignment-uniqueness rule, applied here to the equivalent
// question one domain over: "a ServiceEquipment record may have only one
// active Access Attachment" (this milestone's goal 2), which requires
// comparing the record being created/updated against what else is
// already persisted — something AccessAttachment.Validate (a pure,
// no-dependency function — see internal/accessattachment/validate.go)
// cannot do. That comparison belongs here, the one layer that already
// holds the repository dependency needed to make it.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// AccessAttachmentService is the Access Attachment domain's business
// logic.
//
// It depends only on accessattachment.AccessAttachmentRepository — not
// clock.Clock, for the same reason
// internal/serviceequipment/service.ServiceEquipmentService does not:
// timestamps are already the repository's responsibility, and this
// service has no business rule that needs to reason about "now".
type AccessAttachmentService struct {
	attachments accessattachment.AccessAttachmentRepository
}

// NewAccessAttachmentService builds an AccessAttachmentService.
func NewAccessAttachmentService(attachments accessattachment.AccessAttachmentRepository) *AccessAttachmentService {
	return &AccessAttachmentService{attachments: attachments}
}

// Get retrieves an AccessAttachment record by ID.
func (s *AccessAttachmentService) Get(ctx context.Context, id uuid.UUID) (accessattachment.AccessAttachment, error) {
	return s.attachments.Get(ctx, id)
}

// List returns every AccessAttachment record.
func (s *AccessAttachmentService) List(ctx context.Context) ([]accessattachment.AccessAttachment, error) {
	return s.attachments.List(ctx)
}

// Create validates a, enforces the active-attachment-uniqueness rule,
// and if both pass, persists it.
//
// Field validation happens first, for the same reasoning
// internal/serviceequipment/service.ServiceEquipmentService.Create
// documents: invalid input should never cost even a single database
// round trip, let alone the extra GetActiveByServiceEquipmentID query
// the uniqueness check requires. Only a well-formed, currently-active
// attachment (a.Active(), i.e. RemovedAt == nil — see
// internal/accessattachment/model.go) triggers that check at all: this
// milestone's goal 2 explicitly allows creating an already-historical
// record (RemovalReason/InstalledAt/RemovedAt all pre-filled to record
// an attachment that was, say, removed before this system existed), and
// a record that is not active by definition cannot violate "only one
// active attachment per ServiceEquipment."
func (s *AccessAttachmentService) Create(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	if err := a.Validate(); err != nil {
		return accessattachment.AccessAttachment{}, err
	}
	if a.Active() {
		if err := s.ensureNoActiveAttachment(ctx, a.ServiceEquipmentID, uuid.Nil); err != nil {
			return accessattachment.AccessAttachment{}, err
		}
	}
	return s.attachments.Create(ctx, a)
}

// Update validates a, enforces the active-attachment-uniqueness rule,
// and if both pass, persists the change. See Create for why validation
// and the uniqueness check both happen here rather than elsewhere, and
// for why the check is skipped when a is not active.
//
// Unlike Create, Update passes a.ID as the excluded ID to
// ensureNoActiveAttachment: the row already at a.ID is allowed to be the
// active attachment GetActiveByServiceEquipmentID finds — a caller
// correcting a typo on an already-active row, or reassigning
// ServiceEquipmentID on the very record whose equipment is being
// changed, must not conflict with itself.
func (s *AccessAttachmentService) Update(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	if err := a.Validate(); err != nil {
		return accessattachment.AccessAttachment{}, err
	}
	if a.Active() {
		if err := s.ensureNoActiveAttachment(ctx, a.ServiceEquipmentID, a.ID); err != nil {
			return accessattachment.AccessAttachment{}, err
		}
	}
	return s.attachments.Update(ctx, a)
}

// Delete removes the AccessAttachment record identified by id.
func (s *AccessAttachmentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.attachments.Delete(ctx, id)
}

// ensureNoActiveAttachment implements this milestone's goal 2 rule:
// serviceEquipmentID may have at most one active (RemovedAt == nil)
// AccessAttachment record at a time. excludeID is the ID of the record
// being written (uuid.Nil for Create, where no such record exists yet);
// if GetActiveByServiceEquipmentID finds an active attachment whose ID is
// not excludeID, that is a real conflict — some other row already claims
// this equipment.
//
// apperror.KindNotFound from GetActiveByServiceEquipmentID means exactly
// what it says: serviceEquipmentID currently has no active attachment,
// so there is nothing to conflict with. Any other error (e.g.
// apperror.KindInternal) is propagated as-is rather than swallowed — a
// failed lookup must not be silently treated as "no conflict," which
// would let the uniqueness rule be bypassed by an infrastructure
// failure. Mirrors
// internal/serviceequipment/service.ServiceEquipmentService.ensureNoActiveAssignment
// exactly.
func (s *AccessAttachmentService) ensureNoActiveAttachment(ctx context.Context, serviceEquipmentID, excludeID uuid.UUID) error {
	active, err := s.attachments.GetActiveByServiceEquipmentID(ctx, serviceEquipmentID)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return nil
		}
		return err
	}
	if active.ID == excludeID {
		return nil
	}
	return apperror.Conflict(fmt.Sprintf("service equipment %s already has an active access attachment", serviceEquipmentID))
}
