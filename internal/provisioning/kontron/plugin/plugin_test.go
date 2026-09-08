package plugin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	coreplugin "github.com/paladindigitalgh/palladium-oss/internal/plugin"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/plugin"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// fakeServiceProfiler scripts ServiceProfileService.Apply/Remove's
// results independently, and records the last call made to each.
type fakeServiceProfiler struct {
	applyIface, removeIface string
	applyErr, removeErr     error
	applyCalls, removeCalls int
	gotService              service.Service
	gotEquip                serviceequipment.ServiceEquipment
}

func (f *fakeServiceProfiler) Apply(_ context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error) {
	f.applyCalls++
	f.gotService, f.gotEquip = svc, equipment
	if f.applyErr != nil {
		return "", f.applyErr
	}
	return f.applyIface, nil
}

func (f *fakeServiceProfiler) Remove(_ context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error) {
	f.removeCalls++
	f.gotService, f.gotEquip = svc, equipment
	if f.removeErr != nil {
		return "", f.removeErr
	}
	return f.removeIface, nil
}

func TestPluginCapabilitiesDeclaresProvisionResumeSuspendDisconnect(t *testing.T) {
	p := plugin.New(&fakeServiceProfiler{})
	want := map[coreplugin.Capability]bool{
		coreplugin.ProvisionService:  true,
		coreplugin.ResumeService:     true,
		coreplugin.SuspendService:    true,
		coreplugin.DisconnectService: true,
	}

	got := p.Capabilities()
	if len(got) != len(want) {
		t.Fatalf("Capabilities() = %v, want exactly %v", got, want)
	}
	for _, c := range got {
		if !want[c] {
			t.Errorf("Capabilities() included unexpected %s", c)
		}
	}
}

func TestPluginExecuteRejectsUnsupportedCapability(t *testing.T) {
	p := plugin.New(&fakeServiceProfiler{})

	_, err := p.Execute(context.Background(), coreplugin.ReprovisionService, coreplugin.Resource{})
	if !errors.Is(err, coreplugin.ErrUnsupportedCapability) {
		t.Fatalf("Execute() error = %v, want ErrUnsupportedCapability", err)
	}
}

func TestPluginExecuteProvisionAndResumeCallApply(t *testing.T) {
	for _, capability := range []coreplugin.Capability{coreplugin.ProvisionService, coreplugin.ResumeService} {
		profiler := &fakeServiceProfiler{applyIface: "xgs/6/3"}
		p := plugin.New(profiler)

		svc := service.Service{ID: uuid.New()}
		equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}

		result, err := p.Execute(context.Background(), capability, coreplugin.Resource{Service: svc, Equipment: equipment})
		if err != nil {
			t.Fatalf("Execute(%s) error = %v, want nil", capability, err)
		}
		if profiler.applyCalls != 1 || profiler.removeCalls != 0 {
			t.Errorf("Execute(%s) called Apply %d time(s) and Remove %d time(s), want 1 and 0", capability, profiler.applyCalls, profiler.removeCalls)
		}
		if result.Metadata["interface"] != "xgs/6/3" {
			t.Errorf("Execute(%s) Metadata[interface] = %v, want %q", capability, result.Metadata["interface"], "xgs/6/3")
		}
		if profiler.gotService.ID != svc.ID || profiler.gotEquip.ID != equipment.ID {
			t.Errorf("Execute(%s) did not pass Resource through unchanged", capability)
		}
	}
}

func TestPluginExecuteSuspendAndDisconnectCallRemove(t *testing.T) {
	for _, capability := range []coreplugin.Capability{coreplugin.SuspendService, coreplugin.DisconnectService} {
		profiler := &fakeServiceProfiler{removeIface: "xgs/6/3"}
		p := plugin.New(profiler)

		equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}

		result, err := p.Execute(context.Background(), capability, coreplugin.Resource{Equipment: equipment})
		if err != nil {
			t.Fatalf("Execute(%s) error = %v, want nil", capability, err)
		}
		if profiler.removeCalls != 1 || profiler.applyCalls != 0 {
			t.Errorf("Execute(%s) called Remove %d time(s) and Apply %d time(s), want 1 and 0", capability, profiler.removeCalls, profiler.applyCalls)
		}
		if result.Metadata["interface"] != "xgs/6/3" {
			t.Errorf("Execute(%s) Metadata[interface] = %v, want %q", capability, result.Metadata["interface"], "xgs/6/3")
		}
	}
}

func TestPluginExecuteReturnsNoOpResultWhenNothingToDo(t *testing.T) {
	p := plugin.New(&fakeServiceProfiler{applyIface: "", removeIface: ""})

	for _, capability := range []coreplugin.Capability{coreplugin.ProvisionService, coreplugin.SuspendService} {
		result, err := p.Execute(context.Background(), capability, coreplugin.Resource{
			Equipment: serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleRouter},
		})
		if err != nil {
			t.Fatalf("Execute(%s) error = %v, want nil", capability, err)
		}
		if result.Message == "" {
			t.Errorf("Execute(%s) returned an empty Result.Message for the no-op case", capability)
		}
	}
}

func TestPluginExecutePropagatesApplyAndRemoveErrors(t *testing.T) {
	wantErr := errors.New("command failed")

	applyProfiler := &fakeServiceProfiler{applyErr: wantErr}
	if _, err := plugin.New(applyProfiler).Execute(context.Background(), coreplugin.ProvisionService, coreplugin.Resource{}); !errors.Is(err, wantErr) {
		t.Fatalf("Execute(ProvisionService) error = %v, want %v", err, wantErr)
	}

	removeProfiler := &fakeServiceProfiler{removeErr: wantErr}
	if _, err := plugin.New(removeProfiler).Execute(context.Background(), coreplugin.SuspendService, coreplugin.Resource{}); !errors.Is(err, wantErr) {
		t.Fatalf("Execute(SuspendService) error = %v, want %v", err, wantErr)
	}
}
