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

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// deviceGetter and deviceUpdater are the seams ServiceEquipmentService
// depends on instead of the full inventory.DeviceRepository, satisfied
// by the real *inventoryservice.DeviceService so its Validate() logic
// runs on this write like every other caller's — the identical pattern
// internal/provisioning/kontron/service.DeauthorizationService already
// establishes for the same kind of cross-domain side effect on a Device.
type deviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
}

type deviceUpdater interface {
	Update(ctx context.Context, d inventory.Device) (inventory.Device, error)
}

// activeCustomerDeviceGetter is the seam ServiceEquipmentService depends
// on instead of the full customerdevice.CustomerDeviceRepository — used
// only by markDeviceUnused, to answer "is this Device still attached to
// a Customer" before reverting it to Unused (see that method's own doc
// comment).
type activeCustomerDeviceGetter interface {
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (customerdevice.CustomerDevice, error)
}

// activeOnuAuthorizationGetter is the seam ServiceEquipmentService depends
// on instead of the full onuauthorization.Repository — used only by
// syncAccessAttachment, to answer "does this Device have a known,
// vendor-authorized position on the access network" (see that method's
// own doc comment).
type activeOnuAuthorizationGetter interface {
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (onuauthorization.OnuAuthorization, error)
}

// accessInterfaceGetter is the seam ServiceEquipmentService depends on
// instead of the full accessinterface.AccessInterfaceRepository — used
// only by syncAccessAttachment, to resolve an OnuAuthorization's raw
// OLTID/Interface pair to the AccessInterface record it corresponds to,
// if Palladium already has one on file. This is a pure lookup by ID and
// string, with no Kontron-specific (or any other vendor-specific)
// parsing: that already happened once, in
// internal/provisioning/kontron/service, when the OnuAuthorization
// record itself was created — see that package's own doc comment.
type accessInterfaceGetter interface {
	GetByOLTIDAndName(ctx context.Context, oltID uuid.UUID, name string) (accessinterface.AccessInterface, error)
}

// accessAttachmentCreator is the seam ServiceEquipmentService depends on
// instead of the full accessattachment.AccessAttachmentRepository,
// satisfied by the real *accessattachmentservice.AccessAttachmentService
// so its own active-attachment-uniqueness validation runs on this write
// like every other caller's.
type accessAttachmentCreator interface {
	Create(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error)
}

// ServiceEquipmentService is the Service Equipment domain's business
// logic.
//
// It depends on serviceequipment.ServiceEquipmentRepository and, for the
// Device-status side effect Create/Update below now carry, deviceGetter/
// deviceUpdater — not clock.Clock, for the same reason
// internal/service/service.ServiceService does not: timestamps are
// already the repository's responsibility, and this service has no
// business rule that needs to reason about "now". customerDevices was
// added alongside internal/customerdevice: a Device's Status now also
// reflects whether it is attached to a Customer directly (see that
// package's own doc comment on why that coupling exists at all), so
// losing its active ServiceEquipment record must not silently mark it
// Unused if it is still sitting at a Customer's premises.
//
// onuAuthorizations, accessInterfaces, and accessAttachments back a
// second side effect Create now carries, alongside markDeviceActive: see
// syncAccessAttachment's own doc comment for why this exists — closing
// the gap where a Device authorized directly on an OLT (bypassing the
// plain New Device form) could be attached to a Service without
// Palladium ever recording where on the access network it actually
// sits, leaving real provisioning with nowhere to send its config short
// of an operator building the Access Network topology by hand.
type ServiceEquipmentService struct {
	equipment         serviceequipment.ServiceEquipmentRepository
	devices           deviceGetter
	devicesSvc        deviceUpdater
	customerDevices   activeCustomerDeviceGetter
	onuAuthorizations activeOnuAuthorizationGetter
	accessInterfaces  accessInterfaceGetter
	accessAttachments accessAttachmentCreator
}

// NewServiceEquipmentService builds a ServiceEquipmentService.
func NewServiceEquipmentService(
	equipment serviceequipment.ServiceEquipmentRepository,
	devices deviceGetter,
	devicesSvc deviceUpdater,
	customerDevices activeCustomerDeviceGetter,
	onuAuthorizations activeOnuAuthorizationGetter,
	accessInterfaces accessInterfaceGetter,
	accessAttachments accessAttachmentCreator,
) *ServiceEquipmentService {
	return &ServiceEquipmentService{
		equipment:         equipment,
		devices:           devices,
		devicesSvc:        devicesSvc,
		customerDevices:   customerDevices,
		onuAuthorizations: onuAuthorizations,
		accessInterfaces:  accessInterfaces,
		accessAttachments: accessAttachments,
	}
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
// if both pass, persists it — then, if the new record is active, marks
// its Device Active (see markDeviceActive): a Device only ever means
// "currently serving a Customer" by being attached to an active
// ServiceEquipment record (see inventory.DeviceStatus's own doc
// comment), and this is the one place a Device gains that attachment.
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
// per device" — nor should it mark its Device Active, since it was never
// really attached now.
//
// There is no cross-repository transaction in this codebase (see
// internal/provisioning/kontron/service.DeauthorizationService's own doc
// comment on the same limitation): if markDeviceActive or
// syncAccessAttachment fails after the ServiceEquipment record has
// already been created, Create returns that error, but the record
// already exists — a caller seeing this error must reconcile the
// Device's status, or its Access Attachment, manually.
func (s *ServiceEquipmentService) Create(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	if err := e.Validate(); err != nil {
		return serviceequipment.ServiceEquipment{}, err
	}
	if e.Active() {
		if err := s.ensureNoActiveAssignment(ctx, e.DeviceID, uuid.Nil); err != nil {
			return serviceequipment.ServiceEquipment{}, err
		}
	}
	created, err := s.equipment.Create(ctx, e)
	if err != nil {
		return serviceequipment.ServiceEquipment{}, err
	}
	if created.Active() {
		if err := s.markDeviceActive(ctx, created.DeviceID); err != nil {
			return created, err
		}
		if err := s.syncAccessAttachment(ctx, created); err != nil {
			return created, err
		}
	}
	return created, nil
}

// Update validates e, enforces the active-assignment-uniqueness rule, and
// if both pass, persists the change — then, if this write is what
// transitions the record from active to removed (RemovedAt going from
// nil to set), marks its Device Unused again (see markDeviceUnused): the
// mirror image of Create's Active side effect. See Create for why
// validation and the uniqueness check both happen here rather than
// elsewhere, and for why the check is skipped when e is not active.
//
// The active-to-removed transition is detected by comparing against the
// record as it stood immediately before this write (fetched via
// s.equipment.Get), not by inspecting e alone: Update is called for
// every edit to a ServiceEquipment record, not only the one that removes
// it (e.g. correcting Description on an already-active or
// already-removed row), and only an active record actually becoming
// removed here should ever push its Device back to Unused.
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

	before, err := s.equipment.Get(ctx, e.ID)
	if err != nil {
		return serviceequipment.ServiceEquipment{}, err
	}

	updated, err := s.equipment.Update(ctx, e)
	if err != nil {
		return serviceequipment.ServiceEquipment{}, err
	}

	if before.Active() && !updated.Active() {
		if err := s.markDeviceUnused(ctx, updated.DeviceID); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

// Delete permanently removes the ServiceEquipment record identified by
// id -- unlike Update marking a record removed (RemovedAt set, the row
// itself preserved), this is a hard delete, the path
// ServiceDetailView.vue's own "Remove Equipment" button actually uses
// (see internal/serviceequipment/httpapi's DELETE route). If id was
// active, its Device is marked Unused here too, exactly as Update does
// for the RemovedAt transition: a Device's Status must reflect reality
// regardless of which of these two removal paths a caller takes (see
// inventory.DeviceStatus's own doc comment) -- there is no cross-
// repository transaction in this codebase (see Create's own doc comment
// on the same limitation), so if markDeviceUnused fails after the
// ServiceEquipment row has already been deleted, Delete returns that
// error, but the row is already gone.
func (s *ServiceEquipmentService) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.equipment.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.equipment.Delete(ctx, id); err != nil {
		return err
	}

	if existing.Active() {
		return s.markDeviceUnused(ctx, existing.DeviceID)
	}
	return nil
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

// markDeviceActive transitions deviceID's Device to
// inventory.DeviceStatusActive, once a new active ServiceEquipment
// record for it has already been persisted (see Create's own doc
// comment on why a failure here cannot be rolled back). Unlike
// markDeviceUnused, this always writes Active — even over a Retired
// Device: a Device genuinely being reattached to serve a Customer again
// is, by definition, back in service, regardless of what its prior
// status recorded.
//
// A no-op if the Device is already Active, avoiding a pointless write.
func (s *ServiceEquipmentService) markDeviceActive(ctx context.Context, deviceID uuid.UUID) error {
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
// inventory.DeviceStatusUnused, once its active ServiceEquipment record
// has already been marked removed (see Update's own doc comment on why
// a failure here cannot be rolled back).
//
// A no-op if the Device is already Unused or Retired — mirroring
// internal/provisioning/kontron/service.DeauthorizationService's
// markDeviceRetired: re-writing an already-Unused Device is pointless,
// and a Retired Device (fully deauthorized from its OLT — see that
// service's own doc comment) must never be silently un-retired just
// because an old, already-superseded ServiceEquipment record for it
// happened to be edited into removed here too.
//
// Also a no-op if deviceID still has an active customerdevice.CustomerDevice
// attachment: losing a Service is not the same as leaving the Customer's
// premises (see internal/customerdevice's own doc comment) — a Device
// physically still sitting there, simply not fulfilling any Service at
// the moment, is not "Unused" by this package's own definition of that
// word.
func (s *ServiceEquipmentService) markDeviceUnused(ctx context.Context, deviceID uuid.UUID) error {
	device, err := s.devices.Get(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.Status == inventory.DeviceStatusUnused || device.Status == inventory.DeviceStatusRetired {
		return nil
	}
	stillAttachedToCustomer, err := s.hasActiveCustomerAttachment(ctx, deviceID)
	if err != nil {
		return err
	}
	if stillAttachedToCustomer {
		return nil
	}
	device.Status = inventory.DeviceStatusUnused
	_, err = s.devicesSvc.Update(ctx, device)
	return err
}

// syncAccessAttachment closes the Access Attachment gap for a Device
// that was authorized directly on an OLT through
// internal/provisioning/kontron/service.AuthorizeAndCreateDeviceService
// (the OLT blacklist "authorize this ONU" flow, not the plain New Device
// form): that path already knows exactly which OLT and interface the
// Device is sitting on — recorded as an onuauthorization.OnuAuthorization
// — and already ensures a matching internal/ponport.PONPort and
// internal/accessinterface.AccessInterface exist for it (see that
// service's own doc comment). Until now, nothing ever created the last
// link, an internal/accessattachment.AccessAttachment connecting the new
// ServiceEquipment to that AccessInterface, so real provisioning
// (internal/accesstopology.Resolver.Locate, which walks
// ServiceEquipment -> AccessAttachment -> AccessInterface -> PONPort to
// find where to send config) had nowhere to look unless an operator
// built that link by hand in the Network workspace first. This is the
// one place a ServiceEquipment gains that attachment automatically.
//
// Both lookups are pure vendor-agnostic reads — a device ID, then an
// OLT ID and a plain interface string — with no parsing of any
// vendor-specific interface format: that already happened once, in the
// plugin that created the OnuAuthorization record in the first place
// (see accessInterfaceGetter's own doc comment). Keeping it that way
// here is what lets this method serve every future plugin's
// OnuAuthorization records identically, not just Kontron's.
//
// A no-op, not an error, if either lookup comes back
// apperror.KindNotFound: most Devices have no OnuAuthorization at all
// (they were never authorized this way), and even one that does may not
// yet have a matching AccessInterface (e.g. the plugin that created it
// predates this feature, or intentionally left topology sync to a
// future step). Either way, this must never block Create — it preserves
// the existing manual fallback path (building the Access Attachment by
// hand in the Network workspace) for every Device Palladium does not yet
// have a known network position for. Any other error is propagated as-
// is rather than swallowed, the same reasoning ensureNoActiveAssignment
// and hasActiveCustomerAttachment give for their own, symmetric checks.
func (s *ServiceEquipmentService) syncAccessAttachment(ctx context.Context, e serviceequipment.ServiceEquipment) error {
	auth, err := s.onuAuthorizations.GetActiveByDeviceID(ctx, e.DeviceID)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return nil
		}
		return err
	}

	iface, err := s.accessInterfaces.GetByOLTIDAndName(ctx, auth.OLTID, auth.Interface)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return nil
		}
		return err
	}

	_, err = s.accessAttachments.Create(ctx, accessattachment.AccessAttachment{
		AccessInterfaceID:  iface.ID,
		ServiceEquipmentID: e.ID,
	})
	return err
}

// hasActiveCustomerAttachment reports whether deviceID currently has an
// active customerdevice.CustomerDevice record. apperror.KindNotFound
// from GetActiveByDeviceID means exactly what it says: no active
// attachment, not an error — any other error is propagated as-is rather
// than swallowed, the same reasoning
// internal/customerdevice/service.CustomerDeviceService.ensureNoActiveServiceEquipment
// gives for its own, symmetric check.
func (s *ServiceEquipmentService) hasActiveCustomerAttachment(ctx context.Context, deviceID uuid.UUID) (bool, error) {
	_, err := s.customerDevices.GetActiveByDeviceID(ctx, deviceID)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
