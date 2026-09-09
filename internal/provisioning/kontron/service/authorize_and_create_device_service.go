package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
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
// Order matters: the OLT command runs first, and the Device is only
// created once it has already succeeded -- the same
// authorize-before-persist ordering AuthorizeONU itself already
// establishes, and the mirror image of DeauthorizationService's
// command-first ordering. There is no cross-repository transaction in
// this codebase (see DeauthorizationService's own doc comment on the
// same limitation): if Create fails after the OLT command has already
// succeeded, the ONU is authorized on the OLT but Palladium still has no
// Device record for it, and the caller must retry creating the Device
// directly (e.g. the plain New Device form, entering the serial number
// manually) rather than re-running authorization, which would fail
// against an ONU the OLT now already considers authorized.
type AuthorizeAndCreateDeviceService struct {
	authorize onuAuthorizer
	devices   deviceCreator
}

// NewAuthorizeAndCreateDeviceService builds an
// AuthorizeAndCreateDeviceService.
func NewAuthorizeAndCreateDeviceService(authorize onuAuthorizer, devices deviceCreator) *AuthorizeAndCreateDeviceService {
	return &AuthorizeAndCreateDeviceService{authorize: authorize, devices: devices}
}

// AuthorizeAndCreateDevice authorizes device.SerialNumber on oltID/port,
// then creates device as a real inventory.Device once that succeeds.
// Returns the created Device and the interface AuthorizeONU assigned it.
func (s *AuthorizeAndCreateDeviceService) AuthorizeAndCreateDevice(ctx context.Context, oltID uuid.UUID, port string, device inventory.Device) (inventory.Device, string, error) {
	iface, err := s.authorize.AuthorizeONU(ctx, oltID, port, device.SerialNumber)
	if err != nil {
		return inventory.Device{}, "", err
	}

	created, err := s.devices.Create(ctx, device)
	if err != nil {
		return inventory.Device{}, iface, classify("ONU was authorized on the OLT, but the device record could not be created", err)
	}

	return created, iface, nil
}
