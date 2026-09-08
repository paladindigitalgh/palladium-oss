package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	provisioningkontron "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// serviceEquipmentGetter is the seam DeauthorizationService depends on
// instead of the full serviceequipment.ServiceEquipmentRepository.
type serviceEquipmentGetter interface {
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (serviceequipment.ServiceEquipment, error)
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
// AccessAttachment and ServiceEquipment for this device removed: after
// this call, the OLT genuinely has no record of this ONU, so Palladium's
// own data must stop claiming it is still attached and in service (see
// CLAUDE.md's "always model the real world"). There is no cross-
// repository transaction in this codebase (see OLTService.Create's own
// doc comment on the same limitation for its port-creation cascade): if
// either Update call fails after the OLT command has already succeeded,
// DeauthorizeONU returns that error, but the OLT-side change has already
// taken effect and cannot be rolled back from here — a caller seeing
// this error must reconcile the two records manually.
type DeauthorizationService struct {
	dial           dialer
	equipment      serviceEquipmentGetter
	equipmentSvc   serviceEquipmentUpdater
	locate         locator
	olts           oltGetter
	models         oltModelGetter
	attachments    accessAttachmentGetter
	attachmentsSvc accessAttachmentUpdater
	clock          clock.Clock
}

// NewDeauthorizationService builds a DeauthorizationService.
func NewDeauthorizationService(
	dial dialer,
	equipment serviceEquipmentGetter,
	equipmentSvc serviceEquipmentUpdater,
	locate locator,
	olts oltGetter,
	models oltModelGetter,
	attachments accessAttachmentGetter,
	attachmentsSvc accessAttachmentUpdater,
	c clock.Clock,
) *DeauthorizationService {
	return &DeauthorizationService{
		dial: dial, equipment: equipment, equipmentSvc: equipmentSvc,
		locate: locate, olts: olts, models: models,
		attachments: attachments, attachmentsSvc: attachmentsSvc, clock: c,
	}
}

// DeauthorizeONU resolves deviceID's active ServiceEquipment and OLT
// interface, confirms the OLT is Kontron-vendored, runs the
// deauthorization command, and — only once that succeeds — marks the
// active AccessAttachment and ServiceEquipment removed. It returns the
// interface deauthorized.
//
// Unlike ServiceProfileService's run (which is called once per active
// equipment item by internal/workflow/engine's loop, so a non-PON role
// is a silent no-op), this is a single explicit action on one specific
// Device a caller already identified by DeviceID: a role other than ONU
// or ONT is a real error, not something to skip past quietly.
func (s *DeauthorizationService) DeauthorizeONU(ctx context.Context, deviceID uuid.UUID) (string, error) {
	equipment, err := s.equipment.GetActiveByDeviceID(ctx, deviceID)
	if err != nil {
		return "", classify("could not load active equipment for device", err)
	}
	if equipment.Role != serviceequipment.EquipmentRoleONU && equipment.Role != serviceequipment.EquipmentRoleONT {
		return "", apperror.Invalid("device is not an ONU or ONT")
	}

	location, err := s.locate.Locate(ctx, equipment.ID)
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

	return location.Interface, nil
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

	equipment.RemovedAt = &now
	if _, err := s.equipmentSvc.Update(ctx, equipment); err != nil {
		return classify("OLT was deauthorized, but the service equipment could not be marked removed", err)
	}
	return nil
}
