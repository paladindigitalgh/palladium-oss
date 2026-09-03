package plugin_test

import (
	"context"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/plugin"
)

type stubPlugin struct {
	name         string
	capabilities []plugin.Capability
}

func (p stubPlugin) Name() string                      { return p.name }
func (p stubPlugin) Vendor() string                    { return "stub" }
func (p stubPlugin) Capabilities() []plugin.Capability { return p.capabilities }
func (p stubPlugin) Execute(context.Context, plugin.Capability, plugin.Resource) (plugin.Result, error) {
	return plugin.Result{Message: p.name}, nil
}

func TestRegistryResolveReturnsFalseWhenUnregistered(t *testing.T) {
	r := plugin.NewDefaultRegistry()

	_, ok := r.Resolve(plugin.ProvisionService)
	if ok {
		t.Fatal("Resolve() ok = true, want false for an unregistered capability")
	}
}

func TestRegistryResolveReturnsRegisteredPlugin(t *testing.T) {
	r := plugin.NewDefaultRegistry()
	p := stubPlugin{name: "a", capabilities: []plugin.Capability{plugin.SuspendService, plugin.ResumeService}}
	r.Register(p)

	got, ok := r.Resolve(plugin.SuspendService)
	if !ok || got.Name() != "a" {
		t.Fatalf("Resolve(SuspendService) = %v, %v, want plugin %q", got, ok, "a")
	}

	got, ok = r.Resolve(plugin.ResumeService)
	if !ok || got.Name() != "a" {
		t.Fatalf("Resolve(ResumeService) = %v, %v, want plugin %q", got, ok, "a")
	}
}

func TestRegistryLastRegistrationWinsPerCapability(t *testing.T) {
	r := plugin.NewDefaultRegistry()
	r.Register(stubPlugin{name: "first", capabilities: []plugin.Capability{plugin.ProvisionService}})
	r.Register(stubPlugin{name: "second", capabilities: []plugin.Capability{plugin.ProvisionService}})

	got, ok := r.Resolve(plugin.ProvisionService)
	if !ok || got.Name() != "second" {
		t.Fatalf("Resolve(ProvisionService) = %v, %v, want plugin %q", got, ok, "second")
	}
}
