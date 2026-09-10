package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	provisioningkontron "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// locator is the seam ServiceProfileService depends on instead of the
// concrete *accesstopology.Resolver — the same narrowing pattern this
// package's dialer interface already establishes for
// *connect.Dialer.
type locator interface {
	Locate(ctx context.Context, serviceEquipmentID uuid.UUID) (accesstopology.Location, error)
}

// oltGetter is the seam ServiceProfileService depends on instead of the
// full olt.OLTRepository.
type oltGetter interface {
	Get(ctx context.Context, id uuid.UUID) (olt.OLT, error)
}

// oltModelGetter is the seam ServiceProfileService depends on instead
// of the full oltmodel.OLTModelRepository.
type oltModelGetter interface {
	Get(ctx context.Context, id uuid.UUID) (oltmodel.OLTModel, error)
}

// profileLister is the seam ServiceProfileService depends on instead of
// the full provisioning.ProvisioningProfileRepository. List, not a
// narrower ListByProduct-shaped method, because no such server-side
// query exists yet — see internal/provisioning's own repository doc
// comment on why callers filter the full list themselves.
type profileLister interface {
	List(ctx context.Context) ([]provisioning.ProvisioningProfile, error)
}

// ServiceProfileService applies or removes a subscriber's own Product-
// specific Kontron service profile on their already-authorized ONU.
//
// Apply is what ProvisionService and ResumeService both need: activating
// a Service and resuming a previously-suspended one are, at the Kontron
// config level, the exact same action — apply the same named profile —
// so both capabilities share this one method (see
// provisioningkontron.Client.ApplyServiceProfile's own doc comment).
// Remove is what SuspendService and DisconnectService both need, for
// the identical reason on the opposite side: both just remove the named
// profile (see provisioningkontron.Client.RemoveServiceProfile). Which
// Capability maps to which Service lifecycle status afterward is
// internal/workflow/engine's own business rule (serviceStatusAfter),
// not this package's concern.
type ServiceProfileService struct {
	dial     dialer
	locate   locator
	olts     oltGetter
	models   oltModelGetter
	profiles profileLister
}

// NewServiceProfileService builds a ServiceProfileService.
func NewServiceProfileService(dial dialer, locate locator, olts oltGetter, models oltModelGetter, profiles profileLister) *ServiceProfileService {
	return &ServiceProfileService{dial: dial, locate: locate, olts: olts, models: models, profiles: profiles}
}

// Apply resolves equipment's current position in the access network,
// confirms it sits on a Kontron OLT, finds the one ProvisioningProfile
// mapping svc's Product to a Kontron service-profile name, and applies
// it. See run's doc comment for the resolution steps and the meaning of
// an empty returned interface.
func (s *ServiceProfileService) Apply(ctx context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error) {
	return s.run(ctx, svc, equipment, func(client *provisioningkontron.Client, iface, profileName string, uniPort int) error {
		return client.ApplyServiceProfile(ctx, iface, profileName, uniPort)
	})
}

// Remove resolves equipment exactly as Apply does, then removes svc's
// Kontron service-profile from it instead of applying it. See run's doc
// comment for the resolution steps and the meaning of an empty returned
// interface.
func (s *ServiceProfileService) Remove(ctx context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error) {
	// uniPort is deliberately unused here: RemoveServiceProfile takes no
	// such argument (see its own doc comment on why apply and remove are
	// not symmetric on the real hardware) — run's command signature stays
	// shared between Apply and Remove regardless, so this closure just
	// ignores the value Apply's needs.
	return s.run(ctx, svc, equipment, func(client *provisioningkontron.Client, iface, profileName string, _ int) error {
		return client.RemoveServiceProfile(ctx, iface, profileName)
	})
}

// run is Apply and Remove's shared body: resolve equipment's location
// and svc's Kontron ProvisioningProfile, dial the OLT, and invoke
// command with the resulting Client — the two callers differ only in
// which single command they run once resolution is done. It returns the
// interface command was run against, and "" (with a nil error) when
// equipment's Role means neither Apply nor Remove has anything to do
// for it.
//
// A Service's ServiceEquipment records are not all PON-attached: a
// Router, WiFiAccessPoint, or UPS can be equipment on the same Service,
// and internal/workflow/engine's Execute loop calls the one Plugin
// registered for a Capability once per active equipment record
// uniformly, regardless of Role (see internal/plugin.Registry's own doc
// comment: exactly one Plugin per Capability, globally). Only
// EquipmentRoleONU and EquipmentRoleONT are this Plugin's concern;
// anything else is a successful no-op, not a failure.
func (s *ServiceProfileService) run(
	ctx context.Context,
	svc service.Service,
	equipment serviceequipment.ServiceEquipment,
	command func(client *provisioningkontron.Client, iface, profileName string, uniPort int) error,
) (string, error) {
	if equipment.Role != serviceequipment.EquipmentRoleONU && equipment.Role != serviceequipment.EquipmentRoleONT {
		return "", nil
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
		return "", apperror.Invalid(fmt.Sprintf("equipment %s is attached to a %s OLT, not Kontron", equipment.ID, model.Vendor))
	}

	profiles, err := s.profiles.List(ctx)
	if err != nil {
		return "", classify("could not load provisioning profiles", err)
	}
	profile, err := findKontronProfile(profiles, svc.ProductID)
	if err != nil {
		return "", err
	}

	shell, err := s.dial.Dial(ctx, location.OLTID)
	if err != nil {
		return "", classify("could not reach OLT", err)
	}
	defer func() { _ = shell.Close() }()

	if err := command(provisioningkontron.NewClient(shell), location.Interface, profile.ProfileName, equipment.UNIPort); err != nil {
		return "", classify("command failed", err)
	}

	return location.Interface, nil
}

// findKontronProfile returns the one ProvisioningProfile in profiles
// mapping productID to a Kontron service-profile name. It is an error
// for zero or more than one profile to match: zero means this Product
// was never configured for Kontron, and more than one violates
// internal/provisioning's own documented "one profile per Product+
// Vendor" invariant, which is not enforced at the database layer.
func findKontronProfile(profiles []provisioning.ProvisioningProfile, productID uuid.UUID) (provisioning.ProvisioningProfile, error) {
	var matches []provisioning.ProvisioningProfile
	for _, p := range profiles {
		if p.ProductID == productID && p.Vendor == oltmodel.VendorKontron.String() {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return provisioning.ProvisioningProfile{}, apperror.NotFound(fmt.Sprintf("no Kontron ProvisioningProfile for product %s", productID))
	case 1:
		return matches[0], nil
	default:
		return provisioning.ProvisioningProfile{}, apperror.Conflict(fmt.Sprintf("multiple Kontron ProvisioningProfiles for product %s", productID))
	}
}
