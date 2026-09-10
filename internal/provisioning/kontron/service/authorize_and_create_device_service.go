package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
	provisioningkontron "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron"
)

// onuAuthorizer is the seam AuthorizeAndCreateDeviceService depends on
// instead of the concrete *AuthorizationService, satisfied by it exactly
// (AuthorizeONU's own signature). This composes AuthorizationService's
// existing dial/read/authorize sequence rather than duplicating it.
type onuAuthorizer interface {
	AuthorizeONU(ctx context.Context, oltID uuid.UUID, port, serialNumber string) (string, error)
}

// deviceCreator is the seam AuthorizeAndCreateDeviceService depends on
// instead of the full inventory.DeviceRepository, satisfied by the real
// *inventoryservice.DeviceService so its Validate() logic runs on this
// write like every other caller's.
type deviceCreator interface {
	Create(ctx context.Context, device inventory.Device) (inventory.Device, error)
}

// onuAuthorizationCreator is the seam AuthorizeAndCreateDeviceService
// depends on instead of the full onuauthorization.Repository, satisfied
// by the real *onuauthorizationservice.OnuAuthorizationService so its
// Validate() logic runs on this write like every other caller's.
type onuAuthorizationCreator interface {
	Create(ctx context.Context, authorization onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error)
}

// ponPortFinderCreator is the seam AuthorizeAndCreateDeviceService
// depends on instead of the full ponport.PONPortRepository, satisfied by
// the real *ponportservice.PONPortService so its Validate() logic runs
// on this write like every other caller's.
type ponPortFinderCreator interface {
	GetByOLTIDAndPortNumber(ctx context.Context, oltID uuid.UUID, portNumber int) (ponport.PONPort, error)
	Create(ctx context.Context, port ponport.PONPort) (ponport.PONPort, error)
}

// accessInterfaceFinderCreator is the seam AuthorizeAndCreateDeviceService
// depends on instead of the full accessinterface.AccessInterfaceRepository,
// satisfied by the real *accessinterfaceservice.AccessInterfaceService so
// its Validate() logic runs on this write like every other caller's.
type accessInterfaceFinderCreator interface {
	GetByOLTIDAndName(ctx context.Context, oltID uuid.UUID, name string) (accessinterface.AccessInterface, error)
	Create(ctx context.Context, iface accessinterface.AccessInterface) (accessinterface.AccessInterface, error)
}

// AuthorizeAndCreateDeviceService merges what used to be two separate,
// independently skippable operator actions -- authorizing a blacklisted
// ONU on its OLT (AuthorizationService.AuthorizeONU), then separately
// remembering to create a Palladium Device record for it through the
// plain New Device form -- into one action triggered by a single UI
// submit. Before this existed, an operator could authorize an ONU,
// dismiss or abandon the follow-up "New Device" form, and end up with an
// ONU live and serving traffic on the OLT with no corresponding Device
// record anywhere in Palladium: authorized-on-the-network and
// tracked-in-inventory silently diverged.
//
// It also persists an onuauthorization.OnuAuthorization record alongside
// the Device (see that package's own doc comment for the full
// motivation): without it, a Device created this way could never be
// deauthorized again, because DeauthorizationService had no way to learn
// which OLT/interface to run the command against for a Device that was
// never assigned to a Service. This record is what lets
// DeauthorizationService fall back to resolving that directly.
//
// Order matters: the OLT command runs first, then the Device, then the
// OnuAuthorization record -- each step only runs once the previous one
// has already succeeded, the same authorize-before-persist ordering
// AuthorizeONU itself already establishes, and the mirror image of
// DeauthorizationService's command-first ordering. There is no
// cross-repository transaction in this codebase (see
// DeauthorizationService's own doc comment on the same limitation): if
// either persistence step fails after an earlier one has already
// succeeded, the caller must reconcile manually -- e.g. a Device that
// exists but has no OnuAuthorization record is exactly the pre-existing
// "authorized but can never be deauthorized" state this package closes
// for the normal path, recoverable only by creating that record directly.
type AuthorizeAndCreateDeviceService struct {
	authorize      onuAuthorizer
	devices        deviceCreator
	authorizations onuAuthorizationCreator
	ponPorts       ponPortFinderCreator
	interfaces     accessInterfaceFinderCreator
	clock          clock.Clock
}

// NewAuthorizeAndCreateDeviceService builds an
// AuthorizeAndCreateDeviceService.
func NewAuthorizeAndCreateDeviceService(
	authorize onuAuthorizer,
	devices deviceCreator,
	authorizations onuAuthorizationCreator,
	ponPorts ponPortFinderCreator,
	interfaces accessInterfaceFinderCreator,
	c clock.Clock,
) *AuthorizeAndCreateDeviceService {
	return &AuthorizeAndCreateDeviceService{
		authorize:      authorize,
		devices:        devices,
		authorizations: authorizations,
		ponPorts:       ponPorts,
		interfaces:     interfaces,
		clock:          c,
	}
}

// AuthorizeAndCreateDevice authorizes device.SerialNumber on oltID/port,
// creates device as a real inventory.Device once that succeeds, and then
// records an onuauthorization.OnuAuthorization for it. Returns the
// created Device and the interface AuthorizeONU assigned it.
func (s *AuthorizeAndCreateDeviceService) AuthorizeAndCreateDevice(ctx context.Context, oltID uuid.UUID, port string, device inventory.Device) (inventory.Device, string, error) {
	iface, err := s.authorize.AuthorizeONU(ctx, oltID, port, device.SerialNumber)
	if err != nil {
		return inventory.Device{}, "", err
	}

	created, err := s.devices.Create(ctx, device)
	if err != nil {
		return inventory.Device{}, iface, classify("ONU was authorized on the OLT, but the device record could not be created", err)
	}

	if _, err := s.authorizations.Create(ctx, onuauthorization.OnuAuthorization{
		DeviceID:     created.ID,
		OLTID:        oltID,
		Interface:    iface,
		AuthorizedAt: s.clock.Now(),
	}); err != nil {
		return created, iface, classify("ONU was authorized and the device record was created, but its network authorization record could not be saved -- this device may not be deauthorizable through Palladium until that is fixed manually", err)
	}

	if err := s.syncAccessTopology(ctx, oltID, iface); err != nil {
		return created, iface, classify("ONU was authorized, the device record was created, and its network authorization record was saved, but its Access Network topology could not be synced -- this device may need an Access Attachment built by hand in the Network workspace before it can be provisioned", err)
	}

	return created, iface, nil
}

// syncAccessTopology ensures a PONPort and AccessInterface already exist
// in Palladium for oltID/iface, creating whichever is missing. This is
// what lets internal/serviceequipment/service.ServiceEquipmentService's
// own syncAccessAttachment (see that method's own doc comment) find a
// vendor-agnostic AccessInterface to attach a Device's ServiceEquipment
// to later, with no operator ever having to build that topology by hand
// in the Network workspace for a Device authorized this way -- the
// user's own stated requirement for this feature.
//
// Parsing iface (a Kontron-shaped string like "xgs/1/1") is deliberately
// done here, in this vendor-specific package, not in
// internal/serviceequipment or internal/accessinterface: see
// kontron.ParsePortNumber's own doc comment, and CLAUDE.md's Plugin
// Philosophy ("everything vendor-specific belongs in plugins"). Every
// AccessInterface this creates is XGSPON: every interface this plugin's
// AuthorizeONU can ever return is "xgs"-prefixed (see that method's own
// doc comment) -- there is no other technology for this plugin to
// report.
//
// A found-or-created PONPort/AccessInterface pair is idempotent: a
// second Device authorized on the same port reuses the same PONPort
// record, and a second ONU index on an already-known port reuses nothing
// (each ONU index is its own AccessInterface), so repeated calls for
// devices sharing a port never create duplicates of the port itself.
func (s *AuthorizeAndCreateDeviceService) syncAccessTopology(ctx context.Context, oltID uuid.UUID, iface string) error {
	portNumber, err := provisioningkontron.ParsePortNumber(iface)
	if err != nil {
		return err
	}

	port, err := s.ponPorts.GetByOLTIDAndPortNumber(ctx, oltID, portNumber)
	if err != nil {
		if !apperror.Is(err, apperror.KindNotFound) {
			return err
		}
		port, err = s.ponPorts.Create(ctx, ponport.PONPort{OLTID: oltID, PortNumber: portNumber})
		if err != nil {
			return err
		}
	}

	if _, err := s.interfaces.GetByOLTIDAndName(ctx, oltID, iface); err != nil {
		if !apperror.Is(err, apperror.KindNotFound) {
			return err
		}
		if _, err := s.interfaces.Create(ctx, accessinterface.AccessInterface{
			PONPortID:  port.ID,
			Technology: accessinterface.TechnologyXGSPON,
			Name:       iface,
			Status:     accessinterface.StatusActive,
		}); err != nil {
			return err
		}
	}

	return nil
}
