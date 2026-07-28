package diagnostics_test

import (
	"context"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
)

// stubDiagnostic has no real behavior — it exists solely to prove
// Diagnostic is satisfiable with a sane, consistent method shape and to
// give Registry tests something concrete to register and look back up.
// Mirrors internal/provisioning/connectors/registry_test.go's
// stubConnector for the same reason. marker lets a test distinguish two
// stubDiagnostic values that share the same name (Registry is keyed by
// name, so a second registration under the same name must be
// distinguishable some other way to prove it replaced, not merged with,
// the first).
type stubDiagnostic struct {
	name   string
	marker string
}

func (d stubDiagnostic) Name() string { return d.name }
func (d stubDiagnostic) Run(context.Context, diagnostics.Request) (*diagnostics.Result, error) {
	return &diagnostics.Result{Name: d.name, Sections: []diagnostics.Section{{Name: d.marker}}}, nil
}

var _ diagnostics.Diagnostic = stubDiagnostic{}

func TestDefaultRegistryGetReturnsFalseWhenNoDiagnosticRegistered(t *testing.T) {
	registry := diagnostics.NewDefaultRegistry()

	_, ok := registry.Get("Basic ONU Check")
	if ok {
		t.Error("Get() ok = true, want false for a name with nothing registered")
	}
}

func TestDefaultRegistryRegisterAndGet(t *testing.T) {
	registry := diagnostics.NewDefaultRegistry()
	diagnostic := stubDiagnostic{name: "Basic ONU Check"}

	registry.Register(diagnostic)

	got, ok := registry.Get("Basic ONU Check")
	if !ok {
		t.Fatal("Get() ok = false, want true after Register()")
	}
	if got.Name() != diagnostic.Name() {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), diagnostic.Name())
	}
}

// TestDefaultRegistryGetIsScopedToName proves a diagnostic registered
// under one name is not returned for a different name — Registry
// answers "who handles this name" specifically, not "is there any
// diagnostic at all."
func TestDefaultRegistryGetIsScopedToName(t *testing.T) {
	registry := diagnostics.NewDefaultRegistry()
	registry.Register(stubDiagnostic{name: "Basic ONU Check"})

	_, ok := registry.Get("Advanced ONU Check")
	if ok {
		t.Error("Get(\"Advanced ONU Check\") ok = true, want false; only \"Basic ONU Check\" is registered")
	}
}

// TestDefaultRegistryRegisterUsesDiagnosticsOwnName proves Register
// derives the registration key from diagnostic.Name() itself, with no
// separate key parameter — the distinguishing behavior from
// connectors.Registry.Register documented in registry.go.
func TestDefaultRegistryRegisterUsesDiagnosticsOwnName(t *testing.T) {
	registry := diagnostics.NewDefaultRegistry()
	registry.Register(stubDiagnostic{name: "Custom Check"})

	_, ok := registry.Get("Custom Check")
	if !ok {
		t.Fatal("Get(\"Custom Check\") ok = false, want true; Register must key by diagnostic.Name()")
	}
}

// TestDefaultRegistryRegisterReplacesPriorDiagnosticForSameName proves
// the documented "last write wins" semantics: registering a second
// Diagnostic under a name already in use replaces the first, rather
// than erroring or keeping both.
func TestDefaultRegistryRegisterReplacesPriorDiagnosticForSameName(t *testing.T) {
	registry := diagnostics.NewDefaultRegistry()
	first := stubDiagnostic{name: "Basic ONU Check", marker: "first"}
	second := stubDiagnostic{name: "Basic ONU Check", marker: "second"}

	registry.Register(first)
	registry.Register(second)

	got, ok := registry.Get("Basic ONU Check")
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	result, err := got.Run(context.Background(), diagnostics.Request{})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if len(result.Sections) != 1 || result.Sections[0].Name != "second" {
		t.Errorf("Run() came from the first-registered diagnostic, want the second (most recently registered)")
	}
}
