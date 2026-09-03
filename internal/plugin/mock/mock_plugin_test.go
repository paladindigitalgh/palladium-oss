package mock_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/plugin"
	"github.com/paladindigitalgh/palladium-oss/internal/plugin/mock"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

func TestMockPluginExecutesEveryDeclaredCapability(t *testing.T) {
	p := mock.NewMockPlugin(nil)
	resource := plugin.Resource{
		Service:   service.Service{ID: uuid.New()},
		Equipment: serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRole("ONU")},
	}

	for _, capability := range p.Capabilities() {
		result, err := p.Execute(context.Background(), capability, resource)
		if err != nil {
			t.Errorf("Execute(%s) error = %v, want nil", capability, err)
		}
		if result.Message == "" {
			t.Errorf("Execute(%s) returned an empty Result.Message", capability)
		}
	}
}

func TestMockPluginRejectsUnsupportedCapability(t *testing.T) {
	p := mock.NewMockPlugin(nil)

	_, err := p.Execute(context.Background(), plugin.Capability("Unknown"), plugin.Resource{})
	if err != plugin.ErrUnsupportedCapability {
		t.Fatalf("Execute() error = %v, want %v", err, plugin.ErrUnsupportedCapability)
	}
}
