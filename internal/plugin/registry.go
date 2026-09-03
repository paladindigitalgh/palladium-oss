package plugin

import "sync"

// Registry looks up which Plugin provides a given Capability, so
// internal/workflow's engine never hard-codes plugin selection — it
// only ever asks "who provides this Capability," never names a vendor
// directly (see docs/06-PLUGIN-ARCHITECTURE.md's Architect's Note:
// "Palladium should never need to ask, 'Is this a Kontron, Nokia,
// Adtran, or MikroTik?' Instead, it asks, 'Which plugin provides the
// capability I need?'").
type Registry interface {
	// Register associates plugin with every Capability it declares.
	// Registering a second Plugin for a Capability another Plugin already
	// claims replaces the first for that Capability — last write wins,
	// the same semantics a plain map would have.
	Register(p Plugin)
	// Resolve returns the Plugin registered for capability, and false if
	// none is.
	Resolve(capability Capability) (Plugin, bool)
}

// DefaultRegistry is Registry's one implementation: an in-memory map,
// guarded by a mutex. In this codebase's production wiring every
// Register call happens once, at startup, before the HTTP server (and
// therefore concurrent Resolve calls) starts.
type DefaultRegistry struct {
	mu      sync.RWMutex
	plugins map[Capability]Plugin
}

var _ Registry = (*DefaultRegistry)(nil)

// NewDefaultRegistry builds an empty DefaultRegistry.
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{plugins: make(map[Capability]Plugin)}
}

// Register implements Registry.
func (r *DefaultRegistry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, capability := range p.Capabilities() {
		r.plugins[capability] = p
	}
}

// Resolve implements Registry.
func (r *DefaultRegistry) Resolve(capability Capability) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[capability]
	return p, ok
}
