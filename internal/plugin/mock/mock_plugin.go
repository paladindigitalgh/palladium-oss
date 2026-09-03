// Package mock provides MockPlugin, a simulated vendor Plugin. It
// performs no real network communication — every operation is an
// in-memory, deterministic success — so operational workflows can be
// exercised end-to-end without real OLT, router, or ONU hardware,
// satisfying docs/06-PLUGIN-ARCHITECTURE.md section 18, "Testing &
// Simulation."
package mock

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/paladindigitalgh/palladium-oss/internal/plugin"
)

// MockPlugin implements plugin.Plugin for every defined Capability. It
// always succeeds; there is no injectable failure mode because nothing
// in this milestone's scope needs one.
type MockPlugin struct {
	logger *slog.Logger
}

var _ plugin.Plugin = (*MockPlugin)(nil)

// NewMockPlugin builds a MockPlugin. logger may be nil, in which case
// slog.Default() is used.
func NewMockPlugin(logger *slog.Logger) *MockPlugin {
	if logger == nil {
		logger = slog.Default()
	}
	return &MockPlugin{logger: logger}
}

// Name implements plugin.Plugin.
func (p *MockPlugin) Name() string { return "mock" }

// Vendor implements plugin.Plugin.
func (p *MockPlugin) Vendor() string { return "Simulated" }

// Capabilities implements plugin.Plugin.
func (p *MockPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{
		plugin.ProvisionService,
		plugin.ReprovisionService,
		plugin.SuspendService,
		plugin.ResumeService,
		plugin.DisconnectService,
		plugin.SynchronizeService,
	}
}

// Execute implements plugin.Plugin. It performs no real work: it logs
// the simulated action and returns a success Result describing what a
// real vendor implementation would have done.
func (p *MockPlugin) Execute(ctx context.Context, capability plugin.Capability, r plugin.Resource) (plugin.Result, error) {
	switch capability {
	case plugin.ProvisionService, plugin.ReprovisionService, plugin.SuspendService,
		plugin.ResumeService, plugin.DisconnectService, plugin.SynchronizeService:
		message := fmt.Sprintf("simulated %s for equipment %s (role %s)", capability, r.Equipment.ID, r.Equipment.Role)
		p.logger.InfoContext(ctx, "mock plugin executed capability",
			"capability", capability,
			"service_id", r.Service.ID,
			"equipment_id", r.Equipment.ID,
			"equipment_role", r.Equipment.Role,
		)
		return plugin.Result{Message: message}, nil
	default:
		return plugin.Result{}, plugin.ErrUnsupportedCapability
	}
}
