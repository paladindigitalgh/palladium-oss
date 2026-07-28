// Package service is the Service Equipment domain's business logic
// layer. It sits between the HTTP layer and the repository layer: HTTP
// handlers never call a repository directly (see
// internal/serviceequipment/httpapi), and repositories never validate or
// otherwise reason about business rules (see
// internal/serviceequipment/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/service/service exactly, including that package's note on why
// nesting a package named "service" inside a domain package whose own
// name also contains "service" (internal/serviceequipment/service) is the
// expected, consistent result of this codebase's per-domain layering
// convention, not a naming accident.
//
// This is the first business logic layer in this codebase to enforce a
// rule beyond "reject invalid input before it reaches the repository":
// goal 2's active-assignment-uniqueness rule — "a device may have only
// one active assignment" — requires comparing the record being
// created/updated against what else is already persisted, which
// Service.Validate (a pure, no-dependency function — see
// internal/serviceequipment/validate.go) cannot do. That comparison
// belongs here, the one layer that already holds the repository
// dependency needed to make it.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// ServiceEquipmentService is the Service Equipment domain's business
// logic.
//
// It depends only on serviceequipment.ServiceEquipmentRepository — not
// clock.Clock, for the same reason internal/service/service.ServiceService
// does not: timestamps are already the repository's responsibility, and
// this service has no business rule that needs to reason about "now".
type ServiceEquipmentService struct {
	equipment serviceequipment.ServiceEquipmentRepository
}

// NewServiceEquipmentService builds a ServiceEquipmentService.
func NewServiceEquipmentService(equipment serviceequipment.ServiceEquipmentRepository) *ServiceEquipmentService {
	return &ServiceEquipmentService{equipment: equipment}
}

// Get retrieves a ServiceEquipment record by ID.
func (s *ServiceEquipmentService) Get(ctx context.Context, id uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	return s.equipment.Get(ctx, id)
}

// List returns every ServiceEquipment record.
func (s *ServiceEquipmentService) List(ctx context.Context) ([]serviceequipment.ServiceEquipment, error) {
	return s.equipment.List(ctx)
}

// Create validates e, enforces the active-assignment-uniqueness rule, and
// if both pass, persists it.
//
// Field validation happens first, for the same reasoning
// internal/service/service.ServiceService.Create documents: invalid input
// should never cost even a single database round trip, let alone the
// extra GetActiveByDeviceID query the uniqueness check requires. Only a
// well-formed, currently-active assignment (e.Active(), i.e. RemovedAt ==
// nil — see internal/serviceequipment/model.go) triggers that check at
// all: goal 2 explicitly allows creating an already-historical record
// (Description/InstalledAt/RemovedAt all pre-filled to record equipment
// that was, say, removed before this system existed), and a record that
// is not active by definition cannot violate "only one active assignment
// per device."
func (s *ServiceEquipmentService) Create(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	if err := e.Validate(); err != nil {
		return serviceequipment.ServiceEquipment{}, err
	}
	if e.Active() {
		if err := s.ensureNoActiveAssignment(ctx, e.DeviceID, uuid.Nil); err != nil {
			return serviceequipment.ServiceEquipment{}, err
		}
	}
	return s.equipment.Create(ctx, e)
}

// Update validates e, enforces the active-assignment-uniqueness rule, and
// if both pass, persists the change. See Create for why validation and
// the uniqueness check both happen here rather than elsewhere, and for
// why the check is skipped when e is not active.
//
// Unlike Create, Update passes e.ID as the excluded ID to
// ensureNoActiveAssignment: the row already at e.ID is allowed to be the
// active assignment GetActiveByDeviceID finds — a caller correcting a
// typo in Description on an already-active row, or reassigning DeviceID
// on the very record whose device is being changed, must not conflict
// with itself.
func (s *ServiceEquipmentService) Update(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	if err := e.Validate(); err != nil {
		return serviceequipment.ServiceEquipment{}, err
	}
	if e.Active() {
		if err := s.ensureNoActiveAssignment(ctx, e.DeviceID, e.ID); err != nil {
			return serviceequipment.ServiceEquipment{}, err
		}
	}
	return s.equipment.Update(ctx, e)
}

// Delete removes the ServiceEquipment record identified by id.
func (s *ServiceEquipmentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.equipment.Delete(ctx, id)
}

// ensureNoActiveAssignment implements goal 2's rule: deviceID may have at
// most one active (RemovedAt == nil) ServiceEquipment record at a time.
// excludeID is the ID of the record being written (uuid.Nil for Create,
// where no such record exists yet); if GetActiveByDeviceID finds an
// active assignment whose ID is not excludeID, that is a real conflict —
// some other row already claims this device.
//
// apperror.KindNotFound from GetActiveByDeviceID means exactly what it
// says: deviceID currently has no active assignment, so there is nothing
// to conflict with. Any other error (e.g. apperror.KindInternal) is
// propagated as-is rather than swallowed — a failed lookup must not be
// silently treated as "no conflict," which would let the uniqueness rule
// be bypassed by an infrastructure failure.
func (s *ServiceEquipmentService) ensureNoActiveAssignment(ctx context.Context, deviceID, excludeID uuid.UUID) error {
	active, err := s.equipment.GetActiveByDeviceID(ctx, deviceID)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return nil
		}
		return err
	}
	if active.ID == excludeID {
		return nil
	}
	return apperror.Conflict(fmt.Sprintf("device %s already has an active service equipment assignment", deviceID))
}
