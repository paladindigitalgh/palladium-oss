package accesstopology

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// fakeAttachmentGetter, fakeInterfaceGetter, and fakePortGetter are
// in-memory attachmentGetter / interfaceGetter / portGetter
// implementations — Resolver's tests exercise the exact order and
// short-circuiting of its three lookups without a real repository or
// database, mirroring internal/olt/connect/dialer_test.go's own fakes
// exactly.
type fakeAttachmentGetter struct {
	attachment accessattachment.AccessAttachment
	err        error
	gotID      uuid.UUID
}

func (f *fakeAttachmentGetter) GetActiveByServiceEquipmentID(_ context.Context, serviceEquipmentID uuid.UUID) (accessattachment.AccessAttachment, error) {
	f.gotID = serviceEquipmentID
	if f.err != nil {
		return accessattachment.AccessAttachment{}, f.err
	}
	return f.attachment, nil
}

type fakeInterfaceGetter struct {
	iface  accessinterface.AccessInterface
	err    error
	called bool
	gotID  uuid.UUID
}

func (f *fakeInterfaceGetter) Get(_ context.Context, id uuid.UUID) (accessinterface.AccessInterface, error) {
	f.called = true
	f.gotID = id
	if f.err != nil {
		return accessinterface.AccessInterface{}, f.err
	}
	return f.iface, nil
}

type fakePortGetter struct {
	port   ponport.PONPort
	err    error
	called bool
	gotID  uuid.UUID
}

func (f *fakePortGetter) Get(_ context.Context, id uuid.UUID) (ponport.PONPort, error) {
	f.called = true
	f.gotID = id
	if f.err != nil {
		return ponport.PONPort{}, f.err
	}
	return f.port, nil
}

// resolvedChain builds a consistent AccessAttachment -> AccessInterface
// -> PONPort chain (every ID cross-referenced correctly) for tests that
// need the happy path.
func resolvedChain() (accessattachment.AccessAttachment, accessinterface.AccessInterface, ponport.PONPort) {
	interfaceID := uuid.New()
	oltID := uuid.New()

	attachment := accessattachment.AccessAttachment{ID: uuid.New(), AccessInterfaceID: interfaceID}
	iface := accessinterface.AccessInterface{ID: interfaceID, Name: "xgs/1/1", PONPortID: uuid.New()}
	port := ponport.PONPort{ID: iface.PONPortID, OLTID: oltID}
	return attachment, iface, port
}

func TestLocateResolvesFullChain(t *testing.T) {
	attachment, iface, port := resolvedChain()
	attachments := &fakeAttachmentGetter{attachment: attachment}
	interfaces := &fakeInterfaceGetter{iface: iface}
	ports := &fakePortGetter{port: port}

	r := NewResolver(attachments, interfaces, ports)

	serviceEquipmentID := uuid.New()
	got, err := r.Locate(context.Background(), serviceEquipmentID)
	if err != nil {
		t.Fatalf("Locate() = %v", err)
	}

	if attachments.gotID != serviceEquipmentID {
		t.Errorf("attachments.GetActiveByServiceEquipmentID called with %v, want %v", attachments.gotID, serviceEquipmentID)
	}
	if interfaces.gotID != attachment.AccessInterfaceID {
		t.Errorf("interfaces.Get called with %v, want %v", interfaces.gotID, attachment.AccessInterfaceID)
	}
	if ports.gotID != iface.PONPortID {
		t.Errorf("ports.Get called with %v, want %v", ports.gotID, iface.PONPortID)
	}

	want := Location{OLTID: port.OLTID, Interface: "xgs/1/1"}
	if got != want {
		t.Errorf("Locate() = %+v, want %+v", got, want)
	}
}

func TestLocatePropagatesAttachmentNotFound(t *testing.T) {
	notFoundErr := apperror.NotFound("no active attachment for this service equipment")
	attachments := &fakeAttachmentGetter{err: notFoundErr}
	interfaces := &fakeInterfaceGetter{}
	ports := &fakePortGetter{}

	r := NewResolver(attachments, interfaces, ports)

	_, err := r.Locate(context.Background(), uuid.New())
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("Locate() error = %v, want %v", err, notFoundErr)
	}
	if interfaces.called {
		t.Error("interfaces.Get was called; it must never be reached when the attachment lookup fails")
	}
	if ports.called {
		t.Error("ports.Get was called; it must never be reached when the attachment lookup fails")
	}
}

func TestLocatePropagatesInterfaceNotFound(t *testing.T) {
	attachment, _, _ := resolvedChain()
	notFoundErr := apperror.NotFound("access interface not found")
	attachments := &fakeAttachmentGetter{attachment: attachment}
	interfaces := &fakeInterfaceGetter{err: notFoundErr}
	ports := &fakePortGetter{}

	r := NewResolver(attachments, interfaces, ports)

	_, err := r.Locate(context.Background(), uuid.New())
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("Locate() error = %v, want %v", err, notFoundErr)
	}
	if ports.called {
		t.Error("ports.Get was called; it must never be reached when the interface lookup fails")
	}
}

func TestLocatePropagatesPortNotFound(t *testing.T) {
	attachment, iface, _ := resolvedChain()
	notFoundErr := apperror.NotFound("pon port not found")
	attachments := &fakeAttachmentGetter{attachment: attachment}
	interfaces := &fakeInterfaceGetter{iface: iface}
	ports := &fakePortGetter{err: notFoundErr}

	r := NewResolver(attachments, interfaces, ports)

	_, err := r.Locate(context.Background(), uuid.New())
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("Locate() error = %v, want %v", err, notFoundErr)
	}
}
