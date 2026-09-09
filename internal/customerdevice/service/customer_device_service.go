// Package service is the Customer Device domain's business logic layer.
// It sits between the HTTP layer and the repository layer, mirroring
// internal/serviceequipment/service exactly, one domain over: this is
// the second business logic layer in this codebase (after
// ServiceEquipmentService) to enforce a rule beyond "reject invalid
// input before it reaches the repository."
//
// Two such rules apply here:
//
//   - "a Device may be attached to at most one Customer at a time" — the
//     exact shape of ServiceEquipmentService's own
//     active-assignment-uniqueness rule, applied one domain up.
//   - "a Device still fulfilling an active Service cannot be detached
//     from its Customer" — Detaching (Update transitioning Active() from
//     true to false) is blocked, not cascaded, while the Device has an
//     active serviceequipment.ServiceEquipment record: the operator must
//     remove it from the Service first. This is the same fail-loud,
//     FK-restrict-style choice the rest of this codebase makes throughout
//     (e.g. a Location with active Services cannot be removed) rather
//     than silently tearing down a live Service assignment as a side
//     effect of a Customer-level action.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// deviceGetter is the seam CustomerDeviceService depends on instead of
// the full inventory.DeviceRepository, satisfied by the real
// *inventoryservice.DeviceService — the same narrow-interface pattern
// internal/serviceequipment/service.ServiceEquipmentService already
// establishes for a cross-domain read of a Device.
type deviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
}

// activeServiceEquipmentGetter is the seam CustomerDeviceService depends
// on instead of the full
// serviceequipment.ServiceEquipmentRepository/ServiceEquipmentService —
// used only to answer "does this Device currently fulfill an active
// Service," the read Delete's blocking rule needs.
type activeServiceEquipmentGetter interface {
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error)
}

// CustomerDeviceService is the Customer Device domain's business logic.
type CustomerDeviceService struct {
	customerDevices customerdevice.CustomerDeviceRepository
	devices         deviceGetter
	equipment       activeServiceEquipmentGetter
}

// NewCustomerDeviceService builds a CustomerDeviceService.
func NewCustomerDeviceService(
	customerDevices customerdevice.CustomerDeviceRepository,
	devices deviceGetter,
	equipment activeServiceEquipmentGetter,
) *CustomerDeviceService {
	return &CustomerDeviceService{customerDevices: customerDevices, devices: devices, equipment: equipment}
}

// Get retrieves a CustomerDevice record by ID.
func (s *CustomerDeviceService) Get(ctx context.Context, id uuid.UUID) (customerdevice.CustomerDevice, error) {
	return s.customerDevices.Get(ctx, id)
}

// List returns every CustomerDevice record.
func (s *CustomerDeviceService) List(ctx context.Context) ([]customerdevice.CustomerDevice, error) {
	return s.customerDevices.List(ctx)
}

// Create validates cd, enforces the active-assignment-uniqueness rule,
// and — for a well-formed, currently-active attachment — rejects a
// Retired Device (see ensureDeviceNotRetired), then persists it.
//
// Field validation happens first, for the same reasoning
// internal/serviceequipment/service.ServiceEquipmentService.Create
// documents: invalid input should never cost even a single database
// round trip. Only a well-formed, currently-active attachment
// (cd.Active(), i.e. DetachedAt == nil) triggers either check at all —
// a record that is not active by definition cannot violate "attached to
// only one Customer at a time," nor can it place a Retired Device
// anywhere real.
func (s *CustomerDeviceService) Create(ctx context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	if err := cd.Validate(); err != nil {
		return customerdevice.CustomerDevice{}, err
	}
	if cd.Active() {
		if err := s.ensureNoActiveAssignment(ctx, cd.DeviceID, uuid.Nil); err != nil {
			return customerdevice.CustomerDevice{}, err
		}
		if err := s.ensureDeviceNotRetired(ctx, cd.DeviceID); err != nil {
			return customerdevice.CustomerDevice{}, err
		}
	}
	return s.customerDevices.Create(ctx, cd)
}

// Update validates cd, enforces the active-assignment-uniqueness rule,
// and — if this write is what transitions the record from active to
// detached (DetachedAt going from nil to set) — blocks the write while
// the Device still fulfills an active Service (see
// ensureNoActiveServiceEquipment). This is how a Device is detached
// from a Customer: DetachedAt set, the row itself preserved, never a
// Delete (see internal/customerdevice.CustomerDeviceRepository's own
// doc comment on why this domain has none).
//
// The active-to-detached transition is detected by comparing against
// the record as it stood immediately before this write (fetched via
// s.customerDevices.Get), not by inspecting cd alone — the same
// reasoning ServiceEquipmentService.Update's own doc comment gives:
// Update is called for every edit to a CustomerDevice record, not only
// the one that detaches it (e.g. correcting Description), and only an
// active record actually becoming detached here should ever be
// subject to the block.
//
// Unlike Create, Update passes cd.ID as the excluded ID to
// ensureNoActiveAssignment: the row already at cd.ID is allowed to be
// the active attachment GetActiveByDeviceID finds.
func (s *CustomerDeviceService) Update(ctx context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	if err := cd.Validate(); err != nil {
		return customerdevice.CustomerDevice{}, err
	}
	if cd.Active() {
		if err := s.ensureNoActiveAssignment(ctx, cd.DeviceID, cd.ID); err != nil {
			return customerdevice.CustomerDevice{}, err
		}
	}

	before, err := s.customerDevices.Get(ctx, cd.ID)
	if err != nil {
		return customerdevice.CustomerDevice{}, err
	}

	if before.Active() && !cd.Active() {
		if err := s.ensureNoActiveServiceEquipment(ctx, before.DeviceID); err != nil {
			return customerdevice.CustomerDevice{}, err
		}
	}

	return s.customerDevices.Update(ctx, cd)
}

// ensureNoActiveAssignment implements "a Device may be attached to at
// most one Customer at a time." excludeID is the ID of the record being
// written (uuid.Nil for Create, where no such record exists yet); if
// GetActiveByDeviceID finds an active attachment whose ID is not
// excludeID, that is a real conflict — some other row already claims
// this Device. Mirrors
// serviceequipment/service.ServiceEquipmentService.ensureNoActiveAssignment
// exactly.
func (s *CustomerDeviceService) ensureNoActiveAssignment(ctx context.Context, deviceID, excludeID uuid.UUID) error {
	active, err := s.customerDevices.GetActiveByDeviceID(ctx, deviceID)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return nil
		}
		return err
	}
	if active.ID == excludeID {
		return nil
	}
	return apperror.Conflict(fmt.Sprintf("device %s is already attached to another customer", deviceID))
}

// ensureDeviceNotRetired rejects attaching a Retired Device to a
// Customer: a Retired Device has already been fully deauthorized from
// its OLT (see inventory.DeviceStatus's own doc comment) and is not
// somewhere real to physically place, so parking it at a Customer's
// premises would only be misleading.
func (s *CustomerDeviceService) ensureDeviceNotRetired(ctx context.Context, deviceID uuid.UUID) error {
	device, err := s.devices.Get(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.Status == inventory.DeviceStatusRetired {
		return apperror.Invalid("a retired device cannot be attached to a customer")
	}
	return nil
}

// ensureNoActiveServiceEquipment implements Update's detach-blocking
// rule: deviceID must have no active serviceequipment.ServiceEquipment
// record before it can be detached from its Customer.
// apperror.KindNotFound from GetActiveByDeviceID means exactly what it
// says: deviceID currently fulfills no active Service, so there is
// nothing to block on. Any other error is propagated as-is rather than
// swallowed — a failed lookup must not be silently treated as "no
// active service," which would let a Device be detached out from under
// a live Service assignment.
func (s *CustomerDeviceService) ensureNoActiveServiceEquipment(ctx context.Context, deviceID uuid.UUID) error {
	_, err := s.equipment.GetActiveByDeviceID(ctx, deviceID)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return nil
		}
		return err
	}
	return apperror.Conflict(fmt.Sprintf(
		"device %s still fulfills an active service — remove it from the service before detaching", deviceID))
}
