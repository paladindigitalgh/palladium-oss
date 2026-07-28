package diagnostics

import "sync"

// Registry looks up which Diagnostic handles a given name. It exists so
// internal/diagnostics/service.DiagnosticsService never hard-codes which
// Diagnostic implementation backs a given name — the service only ever
// asks a Registry "who handles this," mirroring exactly why
// internal/provisioning/connectors.Registry exists for
// internal/provisioning/engine (see that package's own doc comment).
//
// Registry is an interface, with DefaultRegistry as its one
// implementation here, for the same reason: it lets
// DiagnosticsService's tests supply a fake registry pre-loaded with a
// stub Diagnostic without touching a real implementation, and it keeps
// the door open for a different backing later without changing anything
// that depends on Registry.
//
// Unlike connectors.Registry.Register, which takes an explicit key
// (serviceequipment.EquipmentRole) separate from the Connector itself,
// Register here takes only the Diagnostic — the key is always
// diagnostic.Name(). This is deliberate, not an inconsistency: goal 4 of
// this milestone is explicit that "diagnostics register themselves by
// name," meaning the Diagnostic's own identity is the registration key,
// not some external attribute a caller supplies separately (a
// Diagnostic is not "for" a role or a device type the way a Connector is
// "for" an EquipmentRole — it simply is what its Name says it is).
type Registry interface {
	// Register adds diagnostic to the registry under diagnostic.Name().
	// Registering a second Diagnostic under the same name replaces the
	// first — the last registration for a name wins, the same "last
	// write wins" semantics a plain map would have if callers used one
	// directly.
	Register(diagnostic Diagnostic)

	// Get returns the Diagnostic registered under name, and false if
	// none is. internal/diagnostics/service.DiagnosticsService treats a
	// false return as a real, reportable failure (an
	// apperror.KindNotFound error), not a signal to silently do nothing.
	Get(name string) (Diagnostic, bool)
}

// DefaultRegistry is Registry's one implementation: an in-memory map,
// guarded by a mutex because Register and Get are not otherwise
// guaranteed to be safe for concurrent use together — in this
// codebase's production wiring, every Register call happens once, at
// startup, before the HTTP server (and therefore concurrent Get calls)
// starts, but the mutex costs one field and a few lines to make that a
// guarantee rather than an unstated assumption. Mirrors
// connectors.DefaultRegistry exactly.
type DefaultRegistry struct {
	mu          sync.RWMutex
	diagnostics map[string]Diagnostic
}

// NewDefaultRegistry builds an empty DefaultRegistry. Diagnostics are
// added afterward via Register — there is no constructor parameter for
// initial diagnostics, mirroring connectors.NewDefaultRegistry's own
// reasoning.
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{diagnostics: make(map[string]Diagnostic)}
}

var _ Registry = (*DefaultRegistry)(nil)

// Register implements Registry.
func (r *DefaultRegistry) Register(diagnostic Diagnostic) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.diagnostics[diagnostic.Name()] = diagnostic
}

// Get implements Registry.
func (r *DefaultRegistry) Get(name string) (Diagnostic, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.diagnostics[name]
	return d, ok
}
