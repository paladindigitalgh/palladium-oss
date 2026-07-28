package connectors

import (
	"sync"

	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// Registry looks up which Connector participates for a given piece of
// equipment. It exists specifically so internal/provisioning/engine never
// hard-codes connector selection (goal 3: "Do not hard-code connector
// selection in the engine") — the engine only ever asks a Registry
// "who handles this," it never contains a switch statement naming a
// vendor.
//
// Registry is an interface, with DefaultRegistry as its one
// implementation here, for the same reason Engine is (see
// internal/provisioning/engine's Engine interface): it lets tests supply
// a fake registry pre-loaded with mock connectors without touching real
// connector implementations, and it keeps the door open for a different
// backing (e.g. one that reads registrations from configuration) without
// changing anything that depends on Registry.
type Registry interface {
	// Register associates connector with role, so a future Get(role)
	// returns it. Registering a second Connector for the same role
	// replaces the first — the last registration for a role wins, the
	// same "last write wins" semantics a plain map would have if callers
	// used one directly.
	Register(role serviceequipment.EquipmentRole, connector Connector)

	// Get returns the Connector registered for role, and false if none
	// is. internal/provisioning/engine treats a false return as a real
	// failure (see its own doc comments), not a signal to skip that
	// equipment silently — a piece of equipment with no connector
	// configured for its role is a configuration gap, not something the
	// engine should quietly ignore.
	Get(role serviceequipment.EquipmentRole) (Connector, bool)
}

// DefaultRegistry is Registry's one implementation: an in-memory map,
// guarded by a mutex because Register and Get are not otherwise
// guaranteed to be safe for concurrent use together — in this codebase's
// production wiring, every Register call happens once, at startup, before
// the HTTP server (and therefore concurrent Get calls) starts, but the
// mutex costs one field and a few lines to make that a guarantee rather
// than an unstated assumption.
//
// A future, more elaborate Registry (e.g. one keyed by more than just
// EquipmentRole, or backed by configuration) can implement the same
// Registry interface without internal/provisioning/engine changing at
// all.
type DefaultRegistry struct {
	mu         sync.RWMutex
	connectors map[serviceequipment.EquipmentRole]Connector
}

// NewDefaultRegistry builds an empty DefaultRegistry. Connectors are
// added afterward via Register — there is no constructor parameter for
// initial connectors, since goal 5's dependency-injection list treats
// "Connector Registry" as one already-built dependency the engine
// receives, not something the engine assembles itself.
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{connectors: make(map[serviceequipment.EquipmentRole]Connector)}
}

var _ Registry = (*DefaultRegistry)(nil)

// Register implements Registry.
func (r *DefaultRegistry) Register(role serviceequipment.EquipmentRole, connector Connector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connectors[role] = connector
}

// Get implements Registry.
func (r *DefaultRegistry) Get(role serviceequipment.EquipmentRole) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.connectors[role]
	return c, ok
}
