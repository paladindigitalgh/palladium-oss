package plugin

import (
	"context"
	"errors"
)

// ErrUnsupportedCapability is returned by Execute when called with a
// Capability the Plugin does not declare in Capabilities().
var ErrUnsupportedCapability = errors.New("plugin: unsupported capability")

// Plugin is the interface every vendor-specific implementation satisfies.
// Execute is a single dispatch method, not one method per Capability:
// nothing outside a Plugin implementation ever switches on a Capability
// by name to decide which method to call — the core only ever asks a
// Registry which Plugin provides a Capability, then calls Execute with
// it (see docs/06-PLUGIN-ARCHITECTURE.md, "Capability Model" and
// "Operation Contracts").
type Plugin interface {
	// Name identifies this Plugin for registration, lookup, and audit
	// trails.
	Name() string
	// Vendor identifies the equipment vendor this Plugin implements, for
	// display and diagnostics — it plays no role in Registry lookups,
	// which are keyed by Capability alone (see Registry).
	Vendor() string
	// Capabilities lists every Capability this Plugin can perform.
	Capabilities() []Capability
	// Execute performs capability against r. A Plugin that does not
	// declare capability in Capabilities() returns
	// ErrUnsupportedCapability.
	Execute(ctx context.Context, capability Capability, r Resource) (Result, error)
}
