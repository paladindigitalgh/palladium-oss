package connectors_test

import (
	"context"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/connectors"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// stubConnector has no real behavior — it exists solely to prove
// Connector is satisfiable with a sane, consistent method shape and to
// give Registry tests something concrete to register and look back up.
// internal/provisioning/engine's own tests define a richer mock connector
// for exercising orchestration behavior; this one only needs a name.
type stubConnector struct {
	name string
}

func (c stubConnector) Name() string { return c.name }
func (c stubConnector) Provision(context.Context, connectors.Request) error {
	return connectors.ErrNotImplemented
}
func (c stubConnector) Reprovision(context.Context, connectors.Request) error {
	return connectors.ErrNotImplemented
}
func (c stubConnector) Suspend(context.Context, connectors.Request) error {
	return connectors.ErrNotImplemented
}
func (c stubConnector) Resume(context.Context, connectors.Request) error {
	return connectors.ErrNotImplemented
}
func (c stubConnector) Disconnect(context.Context, connectors.Request) error {
	return connectors.ErrNotImplemented
}
func (c stubConnector) Synchronize(context.Context, connectors.Request) error {
	return connectors.ErrNotImplemented
}

var _ connectors.Connector = stubConnector{}

func TestDefaultRegistryGetReturnsFalseWhenNoConnectorRegistered(t *testing.T) {
	registry := connectors.NewDefaultRegistry()

	_, ok := registry.Get(serviceequipment.EquipmentRoleONU)
	if ok {
		t.Error("Get() ok = true, want false for a role with nothing registered")
	}
}

func TestDefaultRegistryRegisterAndGet(t *testing.T) {
	registry := connectors.NewDefaultRegistry()
	connector := stubConnector{name: "onu-connector"}

	registry.Register(serviceequipment.EquipmentRoleONU, connector)

	got, ok := registry.Get(serviceequipment.EquipmentRoleONU)
	if !ok {
		t.Fatal("Get() ok = false, want true after Register()")
	}
	if got.Name() != connector.Name() {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), connector.Name())
	}
}

// TestDefaultRegistryGetIsScopedToRole proves a connector registered for
// one role is not returned for a different role — Registry answers "who
// handles this role" specifically, not "is there any connector at all."
func TestDefaultRegistryGetIsScopedToRole(t *testing.T) {
	registry := connectors.NewDefaultRegistry()
	registry.Register(serviceequipment.EquipmentRoleONU, stubConnector{name: "onu-connector"})

	_, ok := registry.Get(serviceequipment.EquipmentRoleRouter)
	if ok {
		t.Error("Get(Router) ok = true, want false; only ONU has a registered connector")
	}
}

// TestDefaultRegistryRegisterReplacesPriorConnectorForSameRole proves the
// documented "last write wins" semantics: registering a second Connector
// for a role already in use replaces the first, rather than erroring or
// keeping both.
func TestDefaultRegistryRegisterReplacesPriorConnectorForSameRole(t *testing.T) {
	registry := connectors.NewDefaultRegistry()
	first := stubConnector{name: "first"}
	second := stubConnector{name: "second"}

	registry.Register(serviceequipment.EquipmentRoleONU, first)
	registry.Register(serviceequipment.EquipmentRoleONU, second)

	got, ok := registry.Get(serviceequipment.EquipmentRoleONU)
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if got.Name() != second.Name() {
		t.Errorf("Get().Name() = %q, want %q (the most recently registered connector)", got.Name(), second.Name())
	}
}
