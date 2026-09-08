// Package plugin adapts internal/provisioning/kontron/service's
// ServiceProfileService to internal/plugin's Plugin interface: this is
// the codebase's first real (non-simulated) Plugin, wired into
// internal/workflow's engine via internal/plugin.Registry exactly like
// internal/plugin/mock.MockPlugin, but backed by real SSH commands
// against a Kontron/Iskratel C16 instead of an in-memory success.
//
// It declares plugin.ProvisionService, plugin.ResumeService,
// plugin.SuspendService, and plugin.DisconnectService among
// Capabilities(): applying (Provision/Resume) or removing
// (Suspend/Disconnect) a Product's Kontron service-profile on an ONU is
// the real config-change work built so far (see
// provisioningkontronservice.ServiceProfileService's own doc comment
// for why Provision and Resume share one method, and Suspend and
// Disconnect share another). plugin.ReprovisionService and
// plugin.SynchronizeService remain plugin/mock.MockPlugin's
// responsibility until real Kontron support for those is built —
// registering this Plugin does not touch them (see
// internal/plugin.Registry.Register's own doc comment: a Plugin only
// replaces the prior registration for the Capabilities it declares).
package plugin

import (
	"context"
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/plugin"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// serviceProfiler is the seam Plugin depends on instead of the concrete
// *provisioningkontronservice.ServiceProfileService, the same narrowing
// pattern this codebase uses throughout (see e.g.
// internal/provisioning/kontron/service's own dialer interface).
type serviceProfiler interface {
	Apply(ctx context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error)
	Remove(ctx context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error)
}

// Plugin implements plugin.Plugin for Kontron/Iskratel C16 OLTs.
type Plugin struct {
	serviceProfile serviceProfiler
}

var _ plugin.Plugin = (*Plugin)(nil)

// New builds a Plugin around serviceProfile.
func New(serviceProfile serviceProfiler) *Plugin {
	return &Plugin{serviceProfile: serviceProfile}
}

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "kontron" }

// Vendor implements plugin.Plugin.
func (p *Plugin) Vendor() string { return "Kontron" }

// Capabilities implements plugin.Plugin.
func (p *Plugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{
		plugin.ProvisionService,
		plugin.ResumeService,
		plugin.SuspendService,
		plugin.DisconnectService,
	}
}

// Execute implements plugin.Plugin. plugin.ProvisionService and
// plugin.ResumeService both apply r.Service's Product-specific Kontron
// service-profile to r.Equipment; plugin.SuspendService and
// plugin.DisconnectService both remove it — see
// provisioningkontronservice.ServiceProfileService's own doc comment
// for why activation/resumption and suspension/disconnection are each
// pairs of identical Kontron-side actions.
//
// An empty interface with a nil error from either operation means
// r.Equipment's Role meant this Plugin had nothing to do for it (see
// ServiceProfileService's run method's own doc comment) -- reported as
// a distinct, non-failing Result rather than an error, since the
// engine.go caller treats any error as this Capability's overall
// failure for the Service being acted on.
func (p *Plugin) Execute(ctx context.Context, capability plugin.Capability, r plugin.Resource) (plugin.Result, error) {
	switch capability {
	case plugin.ProvisionService, plugin.ResumeService:
		return p.result(ctx, r, p.serviceProfile.Apply, "applied service profile to %s")
	case plugin.SuspendService, plugin.DisconnectService:
		return p.result(ctx, r, p.serviceProfile.Remove, "removed service profile from %s")
	default:
		return plugin.Result{}, plugin.ErrUnsupportedCapability
	}
}

// result runs action against r, mapping its (interface, error) return
// into a plugin.Result — the shared shape of Execute's two cases, which
// differ only in which method they call and how they phrase success.
func (p *Plugin) result(
	ctx context.Context,
	r plugin.Resource,
	action func(ctx context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error),
	successFormat string,
) (plugin.Result, error) {
	iface, err := action(ctx, r.Service, r.Equipment)
	if err != nil {
		return plugin.Result{}, err
	}
	if iface == "" {
		return plugin.Result{Message: fmt.Sprintf("equipment %s has no PON-attached role; nothing to do", r.Equipment.ID)}, nil
	}
	return plugin.Result{
		Message:  fmt.Sprintf(successFormat, iface),
		Metadata: map[string]any{"interface": iface},
	}, nil
}
