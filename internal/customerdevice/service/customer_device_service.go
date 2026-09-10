// Package service is the Customer Device domain's business logic layer.
// It sits between the HTTP layer and the repository layer, mirroring
// internal/serviceequipment/service exactly, one domain over: this is
// the second business logic layer in this codebase (after
// ServiceEquipmentService) to enforce a rule beyond "reject invalid
// input before it reaches the repository."
//
// Three such rules apply here:
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
//   - a CustomerDevice's optional LocationID, when set, must name a
//     Location belonging to the same CustomerID (see
//     ensureLocationBelongsToCustomer) — the one check here that is not
//     about Active() at all, since LocationID is inert tracking data,
//     never itself gating a state transition the way the two rules above
//     do.
//
// Create/Update also carry a Device-status side effect, the same shape
// ServiceEquipmentService.Create/Update already established one domain
// over: attaching a Device to a Customer marks it
// inventory.DeviceStatusActive (a Device means "in use" by being
// attached to something real, and being physically at a Customer's
// premises already qualifies — it does not need a Service on top of
// that), and detaching it marks it Unused again, unless
// internal/serviceequipment/service.ServiceEquipmentService's own mirror
// check finds the Device still fulfilling an active Service (impossible
// here in practice, since Update's own detach-blocking rule above
// already prevents that combination, but the symmetry is deliberate: two
// independent sources of "this Device is Active" must each be able to
// answer "is the other one still true" before ever reverting to Unused).
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
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

// deviceUpdater is deviceGetter's write-side counterpart, satisfied by
// the real *inventoryservice.DeviceService so its Validate() logic runs
// on this write like every other caller's — mirrors
// internal/serviceequipment/service.ServiceEquipmentService's own
// deviceGetter/deviceUpdater split exactly.
type deviceUpdater interface {
	Update(ctx context.Context, d inventory.Device) (inventory.Device, error)
}

// activeServiceEquipmentGetter is the seam CustomerDeviceService depends
// on instead of the full
// serviceequipment.ServiceEquipmentRepository/ServiceEquipmentService —
// used both by Update's detach-blocking rule and by markDeviceUnused's
// own "is this Device still fulfilling a Service" check.
type activeServiceEquipmentGetter interface {
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error)
}

// locationGetter is the seam CustomerDeviceService depends on instead of
// the full location.LocationRepository, satisfied by the real
// *locationservice.LocationService — the same narrow-interface pattern
// as deviceGetter above. Backs ensureLocationBelongsToCustomer's check
// that a CustomerDevice's optional LocationID, when set, actually names
// one of the same CustomerID's own Locations.
type locationGetter interface {
	Get(ctx context.Context, id uuid.UUID) (location.Location, error)
}

// CustomerDeviceService is the Customer Device domain's business logic.
type CustomerDeviceService struct {
	customerDevices customerdevice.CustomerDeviceRepository
	devices         deviceGetter
	devicesSvc      deviceUpdater
	equipment       activeServiceEquipmentGetter
	locations       locationGetter
}

// NewCustomerDeviceService builds a CustomerDeviceService.
func NewCustomerDeviceService(
	customerDevices customerdevice.CustomerDeviceRepository,
	devices deviceGetter,
	devicesSvc deviceUpdater,
	equipment activeServiceEquipmentGetter,
	locations locationGetter,
) *CustomerDeviceService {
	return &CustomerDeviceService{
		customerDevices: customerDevices,
		devices:         devices,
		devicesSvc:      devicesSvc,
		equipment:       equipment,
		locations:       locations,
	}
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
// (cd.Active(), i.e. DetachedAt == nil) triggers the assignment-
// uniqueness and Retired checks — a record that is not active by
// definition cannot violate "attached to only one Customer at a time,"
// nor can it place a Retired Device anywhere real. ensureLocationBelongsToCustomer
// runs regardless of Active(), though: LocationID is a plain tracking
// field independent of attachment status (see that field's own doc
// comment), so a detached record naming a bogus or foreign Location is
// exactly as wrong as an active one would be.
func (s *CustomerDeviceService) Create(ctx context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error) {
	if err := cd.Validate(); err != nil {
		return customerdevice.CustomerDevice{}, err
	}
	if err := s.ensureLocationBelongsToCustomer(ctx, cd.LocationID, cd.CustomerID); err != nil {
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
	created, err := s.customerDevices.Create(ctx, cd)
	if err != nil {
		return customerdevice.CustomerDevice{}, err
	}
	if created.Active() {
		if err := s.markDeviceActive(ctx, created.DeviceID); err != nil {
			return created, err
		}
	}
	return created, nil
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
// the one that detaches it (e.g. correcting LocationID), and only an
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
	if err := s.ensureLocationBelongsToCustomer(ctx, cd.LocationID, cd.CustomerID); err != nil {
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

	updated, err := s.customerDevices.Update(ctx, cd)
	if err != nil {
		return customerdevice.CustomerDevice{}, err
	}

	if before.Active() && !updated.Active() {
		if err := s.markDeviceUnused(ctx, updated.DeviceID); err != nil {
			return updated, err
		}
	}
	return updated, nil
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

// ensureLocationBelongsToCustomer is a no-op when locationID is nil —
// "not recorded" is this field's own common, legitimate state (see
// CustomerDevice.LocationID's own doc comment), not something to reject.
// When set, it must name a real Location whose own CustomerID matches
// customerID: a CustomerDevice's Location is purely operator-facing
// tracking of where a Device sits (never read by any provisioning or
// billing logic), but tracking a Customer's Device at a different
// Customer's address would only be misleading, the same reasoning
// ensureDeviceNotRetired rejects a Retired Device for.
func (s *CustomerDeviceService) ensureLocationBelongsToCustomer(ctx context.Context, locationID *uuid.UUID, customerID uuid.UUID) error {
	if locationID == nil {
		return nil
	}
	loc, err := s.locations.Get(ctx, *locationID)
	if err != nil {
		return err
	}
	if loc.CustomerID != customerID {
		return apperror.Invalid(fmt.Sprintf("location %s does not belong to customer %s", *locationID, customerID))
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

// markDeviceActive transitions deviceID's Device to
// inventory.DeviceStatusActive, once a new active CustomerDevice
// attachment for it has already been persisted. Mirrors
// internal/serviceequipment/service.ServiceEquipmentService.markDeviceActive
// exactly, including always writing Active even over a Retired Device —
// unreachable in practice here since ensureDeviceNotRetired already
// blocks attaching one, but kept symmetric with that method rather than
// asserting a precondition this method does not itself need to rely on.
//
// A no-op if the Device is already Active, avoiding a pointless write.
func (s *CustomerDeviceService) markDeviceActive(ctx context.Context, deviceID uuid.UUID) error {
	device, err := s.devices.Get(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.Status == inventory.DeviceStatusActive {
		return nil
	}
	device.Status = inventory.DeviceStatusActive
	_, err = s.devicesSvc.Update(ctx, device)
	return err
}

// markDeviceUnused transitions deviceID's Device to
// inventory.DeviceStatusUnused, once its active CustomerDevice
// attachment has already been marked detached. A no-op if the Device is
// already Unused or Retired, mirroring
// ServiceEquipmentService.markDeviceUnused's own guard.
//
// Unlike that method, this one does not need to check for an active
// ServiceEquipment record before reverting to Unused: Update's own
// detach-blocking rule (see ensureNoActiveServiceEquipment, checked
// earlier in the same call) already guarantees this call is only
// reached when no active Service exists for deviceID.
func (s *CustomerDeviceService) markDeviceUnused(ctx context.Context, deviceID uuid.UUID) error {
	device, err := s.devices.Get(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.Status == inventory.DeviceStatusUnused || device.Status == inventory.DeviceStatusRetired {
		return nil
	}
	device.Status = inventory.DeviceStatusUnused
	_, err = s.devicesSvc.Update(ctx, device)
	return err
}
