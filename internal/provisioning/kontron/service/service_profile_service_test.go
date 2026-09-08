package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// fakeLocator scripts accesstopology.Resolver.Locate's result.
type fakeLocator struct {
	location            accesstopology.Location
	err                 error
	gotServiceEquipment uuid.UUID
}

func (f *fakeLocator) Locate(_ context.Context, serviceEquipmentID uuid.UUID) (accesstopology.Location, error) {
	f.gotServiceEquipment = serviceEquipmentID
	if f.err != nil {
		return accesstopology.Location{}, f.err
	}
	return f.location, nil
}

// fakeOLTGetter scripts olt.OLTRepository.Get's result.
type fakeOLTGetter struct {
	olt olt.OLT
	err error
}

func (f *fakeOLTGetter) Get(_ context.Context, _ uuid.UUID) (olt.OLT, error) {
	if f.err != nil {
		return olt.OLT{}, f.err
	}
	return f.olt, nil
}

// fakeOLTModelGetter scripts oltmodel.OLTModelRepository.Get's result.
type fakeOLTModelGetter struct {
	model oltmodel.OLTModel
	err   error
}

func (f *fakeOLTModelGetter) Get(_ context.Context, _ uuid.UUID) (oltmodel.OLTModel, error) {
	if f.err != nil {
		return oltmodel.OLTModel{}, f.err
	}
	return f.model, nil
}

// fakeProfileLister scripts provisioning.ProvisioningProfileRepository.List's result.
type fakeProfileLister struct {
	profiles []provisioning.ProvisioningProfile
	err      error
}

func (f *fakeProfileLister) List(_ context.Context) ([]provisioning.ProvisioningProfile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.profiles, nil
}

func newTestServiceProfileService(dial dialer, locate locator, olts oltGetter, models oltModelGetter, profiles profileLister) *ServiceProfileService {
	return NewServiceProfileService(dial, locate, olts, models, profiles)
}

var kontronOLTModel = oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorKontron, Name: "C16", PONPortCount: 16}

func TestServiceProfileServiceApplySucceedsAndClosesShell(t *testing.T) {
	productID := uuid.New()
	oltID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	svc := service.Service{ID: uuid.New(), ProductID: productID}

	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	profiles := &fakeProfileLister{profiles: []provisioning.ProvisioningProfile{
		{ID: uuid.New(), ProductID: productID, Vendor: "Kontron", ProfileName: "residential-500"},
		{ID: uuid.New(), ProductID: uuid.New(), Vendor: "Kontron", ProfileName: "other-product"},
	}}

	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	iface, err := s.Apply(context.Background(), svc, equipment)
	if err != nil {
		t.Fatalf("Apply() = %v", err)
	}
	if iface != "xgs/6/3" {
		t.Errorf("Apply() interface = %q, want %q", iface, "xgs/6/3")
	}
	if locator.gotServiceEquipment != equipment.ID {
		t.Errorf("locator.gotServiceEquipment = %v, want %v", locator.gotServiceEquipment, equipment.ID)
	}
	if dialer.gotOLTID != oltID {
		t.Errorf("dialer.gotOLTID = %v, want %v", dialer.gotOLTID, oltID)
	}

	want := []string{"configure", "interface xgs/6/3", "service-profile residential-500", "exit", "exit", "save config"}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", shell.calls, want)
	}
	for i, c := range want {
		if shell.calls[i] != c {
			t.Errorf("calls[%d] = %q, want %q", i, shell.calls[i], c)
		}
	}
	if !shell.closeCalled {
		t.Error("shell was not closed")
	}
}

func TestServiceProfileServiceApplyIsNoOpForNonPONRoles(t *testing.T) {
	dialer := &fakeDialer{}
	locator := &fakeLocator{}
	olts := &fakeOLTGetter{}
	models := &fakeOLTModelGetter{}
	profiles := &fakeProfileLister{}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	for _, role := range []serviceequipment.EquipmentRole{
		serviceequipment.EquipmentRoleRouter,
		serviceequipment.EquipmentRoleWiFiAccessPoint,
		serviceequipment.EquipmentRoleUPS,
		serviceequipment.EquipmentRoleGateway,
		serviceequipment.EquipmentRoleOther,
	} {
		equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: role}
		iface, err := s.Apply(context.Background(), service.Service{}, equipment)
		if err != nil {
			t.Errorf("Apply() with role %s error = %v, want nil", role, err)
		}
		if iface != "" {
			t.Errorf("Apply() with role %s interface = %q, want empty", role, iface)
		}
	}

	if locator.gotServiceEquipment != uuid.Nil {
		t.Error("Locate was called for a non-PON role; it should not have been")
	}
}

func TestServiceProfileServiceApplyErrorsForNonKontronOLT(t *testing.T) {
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	nokiaModel := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorNokia, Name: "ISAM"}

	dialer := &fakeDialer{}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: uuid.New(), Interface: "1/1/1"}}
	olts := &fakeOLTGetter{olt: olt.OLT{OLTModelID: nokiaModel.ID}}
	models := &fakeOLTModelGetter{model: nokiaModel}
	profiles := &fakeProfileLister{}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	_, err := s.Apply(context.Background(), service.Service{}, equipment)
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}

func TestServiceProfileServiceApplyErrorsWhenNoProfileMatches(t *testing.T) {
	productID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	svc := service.Service{ProductID: productID}

	dialer := &fakeDialer{}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: uuid.New(), Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	profiles := &fakeProfileLister{profiles: []provisioning.ProvisioningProfile{
		{ID: uuid.New(), ProductID: uuid.New(), Vendor: "Kontron", ProfileName: "other-product"},
	}}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	_, err := s.Apply(context.Background(), svc, equipment)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceProfileServiceApplyErrorsWhenMultipleProfilesMatch(t *testing.T) {
	productID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	svc := service.Service{ProductID: productID}

	dialer := &fakeDialer{}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: uuid.New(), Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	profiles := &fakeProfileLister{profiles: []provisioning.ProvisioningProfile{
		{ID: uuid.New(), ProductID: productID, Vendor: "Kontron", ProfileName: "residential-500"},
		{ID: uuid.New(), ProductID: productID, Vendor: "Kontron", ProfileName: "residential-500-dup"},
	}}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	_, err := s.Apply(context.Background(), svc, equipment)
	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestServiceProfileServiceApplyPropagatesLocateNotFound(t *testing.T) {
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}

	dialer := &fakeDialer{}
	locator := &fakeLocator{err: apperror.NotFound("no active attachment")}
	olts := &fakeOLTGetter{}
	models := &fakeOLTModelGetter{}
	profiles := &fakeProfileLister{}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	_, err := s.Apply(context.Background(), service.Service{}, equipment)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceProfileServiceApplyPropagatesDialFailureAsUnavailable(t *testing.T) {
	productID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	svc := service.Service{ProductID: productID}

	dialer := &fakeDialer{err: errors.New("connection refused")}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: uuid.New(), Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	profiles := &fakeProfileLister{profiles: []provisioning.ProvisioningProfile{
		{ID: uuid.New(), ProductID: productID, Vendor: "Kontron", ProfileName: "residential-500"},
	}}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	_, err := s.Apply(context.Background(), svc, equipment)
	if !apperror.Is(err, apperror.KindUnavailable) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindUnavailable)
	}
}

// TestServiceProfileServiceRemoveSucceedsAndRunsRemovalCommand proves
// Remove resolves equipment identically to Apply (same location, OLT,
// and profile lookup) but runs RemoveServiceProfile's "no service-
// profile" command instead of ApplyServiceProfile's.
func TestServiceProfileServiceRemoveSucceedsAndRunsRemovalCommand(t *testing.T) {
	productID := uuid.New()
	oltID := uuid.New()
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	svc := service.Service{ID: uuid.New(), ProductID: productID}

	shell := &fakeShell{outputs: map[string]string{}}
	dialer := &fakeDialer{shell: shell}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: oltID, Interface: "xgs/6/3"}}
	olts := &fakeOLTGetter{olt: olt.OLT{ID: oltID, OLTModelID: kontronOLTModel.ID}}
	models := &fakeOLTModelGetter{model: kontronOLTModel}
	profiles := &fakeProfileLister{profiles: []provisioning.ProvisioningProfile{
		{ID: uuid.New(), ProductID: productID, Vendor: "Kontron", ProfileName: "residential-500"},
	}}

	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	iface, err := s.Remove(context.Background(), svc, equipment)
	if err != nil {
		t.Fatalf("Remove() = %v", err)
	}
	if iface != "xgs/6/3" {
		t.Errorf("Remove() interface = %q, want %q", iface, "xgs/6/3")
	}

	want := []string{"configure", "interface xgs/6/3", "no service-profile residential-500", "exit", "exit", "save config"}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", shell.calls, want)
	}
	for i, c := range want {
		if shell.calls[i] != c {
			t.Errorf("calls[%d] = %q, want %q", i, shell.calls[i], c)
		}
	}
	if !shell.closeCalled {
		t.Error("shell was not closed")
	}
}

func TestServiceProfileServiceRemoveIsNoOpForNonPONRoles(t *testing.T) {
	dialer := &fakeDialer{}
	locator := &fakeLocator{}
	olts := &fakeOLTGetter{}
	models := &fakeOLTModelGetter{}
	profiles := &fakeProfileLister{}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleRouter}
	iface, err := s.Remove(context.Background(), service.Service{}, equipment)
	if err != nil {
		t.Errorf("Remove() error = %v, want nil", err)
	}
	if iface != "" {
		t.Errorf("Remove() interface = %q, want empty", iface)
	}
	if locator.gotServiceEquipment != uuid.Nil {
		t.Error("Locate was called for a non-PON role; it should not have been")
	}
}

// TestServiceProfileServiceRemoveErrorsForNonKontronOLT proves Remove
// shares Apply's resolution errors via the common run helper — it is
// not a full re-test of every error path (see the Apply-specific tests
// above for those), just confirmation the sharing actually happened.
func TestServiceProfileServiceRemoveErrorsForNonKontronOLT(t *testing.T) {
	equipment := serviceequipment.ServiceEquipment{ID: uuid.New(), Role: serviceequipment.EquipmentRoleONU}
	nokiaModel := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorNokia, Name: "ISAM"}

	dialer := &fakeDialer{}
	locator := &fakeLocator{location: accesstopology.Location{OLTID: uuid.New(), Interface: "1/1/1"}}
	olts := &fakeOLTGetter{olt: olt.OLT{OLTModelID: nokiaModel.ID}}
	models := &fakeOLTModelGetter{model: nokiaModel}
	profiles := &fakeProfileLister{}
	s := newTestServiceProfileService(dialer, locator, olts, models, profiles)

	_, err := s.Remove(context.Background(), service.Service{}, equipment)
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}
