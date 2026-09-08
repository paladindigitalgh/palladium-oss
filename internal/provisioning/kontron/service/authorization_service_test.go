package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron"
)

// fakeShell is an in-memory ssh.Shell recording every command run
// against it, in order, and returning a scripted output per command —
// unlike internal/diagnostics/kontron/service's own fakeShell (which
// only ever sees one RunCommand call per test), AuthorizeONU drives one
// shell through eight commands in a single session (one read, seven
// write), so this fake needs to script all eight.
type fakeShell struct {
	outputs     map[string]string
	err         error
	calls       []string
	closeCalled bool
}

var _ ssh.Shell = (*fakeShell)(nil)

func (f *fakeShell) RunCommand(_ context.Context, command string, _ ...ssh.PagerPrompt) (string, error) {
	f.calls = append(f.calls, command)
	if f.err != nil {
		return "", f.err
	}
	return f.outputs[command], nil
}

func (f *fakeShell) Close() error {
	f.closeCalled = true
	return nil
}

// fakeDialer is an in-memory dialer.
type fakeDialer struct {
	shell    ssh.Shell
	err      error
	gotOLTID uuid.UUID
}

func (f *fakeDialer) Dial(_ context.Context, oltID uuid.UUID) (ssh.Shell, error) {
	f.gotOLTID = oltID
	if f.err != nil {
		return nil, f.err
	}
	return f.shell, nil
}

const onuSummaryWithOnePortInUse = "olt#show onu int all\n" +
	"ONU       Oper     Admin    ONU               Service  Serial\n" +
	"Interface State    State    State             State    Number       Password/Registration Id               IP Address      MAC Address       Description\n" +
	"--------- -------- -------- ----------------  -------- ------------ -------------------------------------- --------------- ----------------- ------------------\n" +
	"xgs/6/1   Up       Enable   Active            Enable   ISKT235A81D8 \"\"                                     10.70.180.4     48:55:41:5A:81:D8 ISKT235A81D8\n"

func TestAuthorizeONUAssignsNextFreeIndexAndClosesShell(t *testing.T) {
	oltID := uuid.New()
	shell := &fakeShell{outputs: map[string]string{
		"show onu interface all": onuSummaryWithOnePortInUse,
	}}
	dialer := &fakeDialer{shell: shell}
	s := NewAuthorizationService(dialer, "iphost")

	iface, err := s.AuthorizeONU(context.Background(), oltID, "xgs/6", "ISKT2308DD88")
	if err != nil {
		t.Fatalf("AuthorizeONU() = %v", err)
	}
	if iface != "xgs/6/2" {
		t.Errorf("AuthorizeONU() interface = %q, want %q (xgs/6/1 already in use)", iface, "xgs/6/2")
	}
	if dialer.gotOLTID != oltID {
		t.Errorf("dialer.gotOLTID = %v, want %v", dialer.gotOLTID, oltID)
	}

	want := []string{
		"show onu interface all",
		"configure",
		"interface xgs/6/2",
		"onu serial-number ISKT2308DD88",
		"service-profile iphost",
		"exit",
		"exit",
		"save config",
	}
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

func TestAuthorizeONUPropagatesDialFailureAsUnavailable(t *testing.T) {
	dialer := &fakeDialer{err: errors.New("connection refused")}
	s := NewAuthorizationService(dialer, "iphost")

	_, err := s.AuthorizeONU(context.Background(), uuid.New(), "xgs/6", "ISKT2308DD88")
	if !apperror.Is(err, apperror.KindUnavailable) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindUnavailable)
	}
}

func TestAuthorizeONUClosesShellEvenWhenCommandFails(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"show onu interface all":         onuSummaryWithOnePortInUse,
		"onu serial-number ISKT2308DD88": "onu_serial_number wrong size!",
	}}
	dialer := &fakeDialer{shell: shell}
	s := NewAuthorizationService(dialer, "iphost")

	_, err := s.AuthorizeONU(context.Background(), uuid.New(), "xgs/6", "ISKT2308DD88")
	if err == nil {
		t.Fatal("AuthorizeONU() error = nil, want an error")
	}
	if !shell.closeCalled {
		t.Error("shell was not closed after a command failure")
	}
	// save config must never have run.
	for _, c := range shell.calls {
		if c == "save config" {
			t.Error("save config ran despite the earlier failure")
		}
	}
}

func TestAuthorizeONUReclassifiesInvalidInterfaceAsInvalid(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"show onu interface all": "", // no rows -> NextFreeIndex returns 1, port itself is fine
	}}
	dialer := &fakeDialer{shell: shell}
	s := NewAuthorizationService(dialer, "iphost")

	// A newline embedded in the serial number is what actually triggers
	// provisioningkontron.ErrInvalidSerialNumber here, since port itself
	// is never attacker-controlled the same way — this proves the
	// reclassification path, not a specific field.
	_, err := s.AuthorizeONU(context.Background(), uuid.New(), "xgs/6", "ISKT2308DD88\nconfigure")
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if !errors.Is(err, kontron.ErrInvalidSerialNumber) {
		t.Errorf("error = %v, want it to wrap kontron.ErrInvalidSerialNumber", err)
	}
}
