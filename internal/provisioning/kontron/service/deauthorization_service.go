package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	provisioningkontron "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// latestServiceEquipmentGetter is the seam DeauthorizationService
// depends on instead of the full
// serviceequipment.ServiceEquipmentRepository. It deliberately asks for
// the device's *latest* ServiceEquipment record rather than its active
// one (contrast internal/provisioning/kontron/service's other seams,
// e.g. AuthorizeONU's, which need an active assignment): a "Remove
// Customer" cascade (internal/customer/removal) already marks
// ServiceEquipment removed to untie a Device from a Customer without
// deleting the Device itself, but the real OLT does not forget the ONU
// just because Palladium's billing-facing record moved on. Deauthorizing
// it for real must still be possible after that point, so it resolves
// from whatever equipment record exists most recently, active or not.
type latestServiceEquipmentGetter interface {
	GetLatestByDeviceID(ctx context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error)
}

// serviceEquipmentUpdater is the seam DeauthorizationService depends on
// for persisting a ServiceEquipment record marked removed — satisfied by
// the real *serviceequipmentservice.ServiceEquipmentService, not the raw
// repository, so its existing Validate() and uniqueness-check logic runs
// on this write exactly as it does on every other caller's (see
// CLAUDE.md's "never duplicate business logic").
type serviceEquipmentUpdater interface {
	Update(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error)
}

// accessAttachmentGetter is the seam DeauthorizationService depends on
// instead of the full accessattachment.AccessAttachmentRepository.
type accessAttachmentGetter interface {
	GetActiveByServiceEquipmentID(ctx context.Context, serviceEquipmentID uuid.UUID) (accessattachment.AccessAttachment, error)
}

// accessAttachmentUpdater is accessAttachmentGetter's write-side
// counterpart, satisfied by the real
// *accessattachmentservice.AccessAttachmentService for the same reason
// serviceEquipmentUpdater is: reuse its Validate() and uniqueness-check
// logic rather than writing straight to the repository.
type accessAttachmentUpdater interface {
	Update(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error)
}

// deviceGetter and deviceUpdater are the seams DeauthorizationService
// depends on instead of the full inventory.DeviceRepository, satisfied
// by the real *inventoryservice.DeviceService so its Validate() logic
// runs on this write like every other caller's. Once an ONU is
// genuinely deauthorized from its OLT, the Device itself is no longer
// installed anywhere real — see markDeviceRetired's own doc comment for
// why this service moves it to DeviceStatusRetired rather than leaving
// it Installed forever.
type deviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
}

type deviceUpdater interface {
	Update(ctx context.Context, d inventory.Device) (inventory.Device, error)
}

// DeauthorizationService fully removes an ONU's authorization from its
// OLT — the mirror image of AuthorizationService, and a step further
// than ServiceProfileService.Remove: Suspend/Disconnect only remove the
// subscriber's service-profile, leaving the ONU reachable on the
// management network, while this removes its base authorization
// entirely (see provisioningkontron.Client.DeauthorizeONU's own doc
// comment). It is triggered from a Device Detail page, not a Service
// action, so it resolves everything starting from a DeviceID rather
// than a ServiceID/ServiceEquipmentID a caller already has in hand.
//
// Once the OLT command succeeds, it also marks the active
// AccessAttachment and ServiceEquipment for this device removed, and the
// Device itself Retired (see markDeviceRetired): after this call, the
// OLT genuinely has no record of this ONU, so Palladium's own data must
// stop claiming it is still attached, in service, and Installed (see
// CLAUDE.md's "always model the real world"). Retiring the Device is
// also what lets it drop out of the Device Collection View's default
// filter — at the user's explicit request, a deauthorized ONU should not
// keep cluttering the default device list. There is no cross-repository
// transaction in this codebase (see OLTService.Create's own doc comment
// on the same limitation for its port-creation cascade): if any of these
// Update calls fails after the OLT command has already succeeded,
// DeauthorizeONU returns that error, but the OLT-side change has already
// taken effect and cannot be rolled back from here — a caller seeing
// this error must reconcile the records manually.
type DeauthorizationService struct {
	dial           dialer
	equipment      latestServiceEquipmentGetter
	equipmentSvc   serviceEquipmentUpdater
	locate         latestLocator
	olts           oltGetter
	models         oltModelGetter
	attachments    accessAttachmentGetter
	attachmentsSvc accessAttachmentUpdater
	devices        deviceGetter
	devicesSvc     deviceUpdater
	clock          clock.Clock
}

// latestLocator is the seam DeauthorizationService depends on instead of
// the concrete *accesstopology.Resolver, mirroring locator
// (service_profile_service.go) but calling LocateLatest instead of
// Locate — see latestServiceEquipmentGetter's own doc comment for why
// deauthorization must resolve location from records that may no longer
// be active.
type latestLocator interface {
	LocateLatest(ctx context.Context, serviceEquipmentID uuid.UUID) (accesstopology.Location, error)
}

// NewDeauthorizationService builds a DeauthorizationService.
func NewDeauthorizationService(
	dial dialer,
	equipment latestServiceEquipmentGetter,
	equipmentSvc serviceEquipmentUpdater,
	locate latestLocator,
	olts oltGetter,
	models oltModelGetter,
	attachments accessAttachmentGetter,
	attachmentsSvc accessAttachmentUpdater,
	devices deviceGetter,
	devicesSvc deviceUpdater,
	c clock.Clock,
) *DeauthorizationService {
	return &DeauthorizationService{
		dial: dial, equipment: equipment, equipmentSvc: equipmentSvc,
		locate: locate, olts: olts, models: models,
		attachments: attachments, attachmentsSvc: attachmentsSvc,
		devices: devices, devicesSvc: devicesSvc, clock: c,
	}
}

// DeauthorizeONU resolves deviceID's latest ServiceEquipment and OLT
// interface — active or already removed, per latestServiceEquipmentGetter's
// doc comment — confirms the OLT is Kontron-vendored, runs the
// deauthorization command, and — only once that succeeds — marks the
// active AccessAttachment and ServiceEquipment removed (a no-op if a
// prior cascade already removed them). It returns the interface
// deauthorized.
//
// Unlike ServiceProfileService's run (which is called once per active
// equipment item by internal/workflow/engine's loop, so a non-PON role
// is a silent no-op), this is a single explicit action on one specific
// Device a caller already identified by DeviceID: a role other than ONU
// or ONT is a real error, not something to skip past quietly.
func (s *DeauthorizationService) DeauthorizeONU(ctx context.Context, deviceID uuid.UUID) (string, error) {
	equipment, err := s.equipment.GetLatestByDeviceID(ctx, deviceID)
	if err != nil {
		return "", classify("could not load equipment for device", err)
	}
	if equipment.Role != serviceequipment.EquipmentRoleONU && equipment.Role != serviceequipment.EquipmentRoleONT {
		return "", apperror.Invalid("device is not an ONU or ONT")
	}

	location, err := s.locate.LocateLatest(ctx, equipment.ID)
	if err != nil {
		return "", classify("could not locate equipment on the access network", err)
	}

	o, err := s.olts.Get(ctx, location.OLTID)
	if err != nil {
		return "", classify("could not load OLT", err)
	}

	model, err := s.models.Get(ctx, o.OLTModelID)
	if err != nil {
		return "", classify("could not load OLT model", err)
	}
	if model.Vendor != oltmodel.VendorKontron {
		return "", apperror.Invalid("equipment is attached to a non-Kontron OLT")
	}

	shell, err := s.dial.Dial(ctx, location.OLTID)
	if err != nil {
		return "", classify("could not reach OLT", err)
	}
	defer func() { _ = shell.Close() }()

	if err := provisioningkontron.NewClient(shell).DeauthorizeONU(ctx, location.Interface); err != nil {
		return "", classify("command failed", err)
	}

	if err := s.markRemoved(ctx, equipment); err != nil {
		return "", err
	}

	if err := s.markDeviceRetired(ctx, deviceID); err != nil {
		return "", err
	}

	return location.Interface, nil
}

// markDeviceRetired transitions deviceID's Device to
// inventory.DeviceStatusRetired, once the OLT-side deauthorization and
// record-marking have already succeeded — see DeauthorizeONU's own doc
// comment on why a failure here cannot be rolled back. A Device that has
// been deauthorized from its OLT is no longer installed anywhere real
// (see CLAUDE.md's "always model the real world"), and Retired is this
// domain's lifecycle state for exactly that: no longer in service, not
// yet disposed of (docs/03-DOMAIN-MODEL.md's Device lifecycle). This is
// what lets a deauthorized ONU drop out of the Device Collection View's
// default (non-Retired) filter without a second, separate action.
//
// A no-op if the Device is already Retired or Disposed — this can
// happen on a retry after a prior partial failure, and re-writing an
// already-terminal status would be pointless.
func (s *DeauthorizationService) markDeviceRetired(ctx context.Context, deviceID uuid.UUID) error {
	device, err := s.devices.Get(ctx, deviceID)
	if err != nil {
		return classify("OLT was deauthorized, but the device could not be loaded to retire it", err)
	}
	if device.Status == inventory.DeviceStatusRetired || device.Status == inventory.DeviceStatusDisposed {
		return nil
	}

	device.Status = inventory.DeviceStatusRetired
	if _, err := s.devicesSvc.Update(ctx, device); err != nil {
		return classify("OLT was deauthorized, but the device could not be marked retired", err)
	}
	return nil
}

// markRemoved sets RemovedAt on the active AccessAttachment (if any) and
// on equipment itself, once the OLT-side deauthorization has already
// succeeded — see DeauthorizeONU's own doc comment on why a failure here
// cannot be rolled back.
func (s *DeauthorizationService) markRemoved(ctx context.Context, equipment serviceequipment.ServiceEquipment) error {
	now := s.clock.Now()

	attachment, err := s.attachments.GetActiveByServiceEquipmentID(ctx, equipment.ID)
	switch {
	case err == nil:
		attachment.RemovedAt = &now
		attachment.RemovalReason = "ONU deauthorized"
		if _, err := s.attachmentsSvc.Update(ctx, attachment); err != nil {
			return classify("OLT was deauthorized, but the access attachment could not be marked removed", err)
		}
	case apperror.Is(err, apperror.KindNotFound):
		// Nothing currently attached; only ServiceEquipment itself needs
		// to be marked removed.
	default:
		return classify("OLT was deauthorized, but the active access attachment could not be loaded", err)
	}

	if equipment.RemovedAt != nil {
		// Already marked removed by an earlier cascade (e.g. Remove
		// Customer un-tying this Device) — nothing left to persist here,
		// and re-writing it would overwrite the original removal time for
		// no reason.
		return nil
	}

	equipment.RemovedAt = &now
	if _, err := s.equipmentSvc.Update(ctx, equipment); err != nil {
		return classify("OLT was deauthorized, but the service equipment could not be marked removed", err)
	}
	return nil
}
