package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// fakeShell is an in-memory ssh.Shell, the same reasoning
// internal/diagnostics/kontron's own fakeShell exists for that
// package's tests.
type fakeShell struct {
	output      string
	err         error
	gotCommand  string
	gotPagers   []ssh.PagerPrompt
	closeCalled bool
	closeErr    error
}

var _ ssh.Shell = (*fakeShell)(nil)

func (f *fakeShell) RunCommand(_ context.Context, command string, pagers ...ssh.PagerPrompt) (string, error) {
	f.gotCommand = command
	f.gotPagers = pagers
	if f.err != nil {
		return "", f.err
	}
	return f.output, nil
}

func (f *fakeShell) Close() error {
	f.closeCalled = true
	return f.closeErr
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

func TestONUSummarySendsExpectedCommandAndClosesShell(t *testing.T) {
	oltID := uuid.New()
	shell := &fakeShell{output: "onu table"}
	dialer := &fakeDialer{shell: shell}
	s := NewKontronService(dialer)

	got, err := s.ONUSummary(context.Background(), oltID)
	if err != nil {
		t.Fatalf("ONUSummary() = %v", err)
	}
	if got != "onu table" {
		t.Errorf("ONUSummary() = %q, want %q", got, "onu table")
	}
	if dialer.gotOLTID != oltID {
		t.Errorf("dial called with %v, want %v", dialer.gotOLTID, oltID)
	}
	if shell.gotCommand != "show onu interface all" {
		t.Errorf("command sent = %q, want %q", shell.gotCommand, "show onu interface all")
	}
	if !shell.closeCalled {
		t.Error("shell was not closed")
	}
}

// TestPerInterfaceMethodsSendExpectedCommands covers every remaining
// KontronService method's delegation and iface plumbing in one table —
// the exact commands themselves are already proven correct at the
// internal/diagnostics/kontron layer; this table exists to prove this
// service calls the right Client method with the right arguments and
// closes the shell.
func TestPerInterfaceMethodsSendExpectedCommands(t *testing.T) {
	const iface = "xgs/1/1"

	cases := []struct {
		name        string
		call        func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error)
		wantCommand string
	}{
		{"ONUStatusSummary", func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error) {
			return s.ONUStatusSummary(ctx, oltID)
		}, "show onu interface all status"},
		{"ONURunningConfig", func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error) {
			return s.ONURunningConfig(ctx, oltID, iface)
		}, "show run xgs/1/1"},
		{"ONUDetail", func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error) {
			return s.ONUDetail(ctx, oltID, iface)
		}, "show onu interface xgs/1/1"},
		{"ONUStatus", func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error) {
			return s.ONUStatus(ctx, oltID, iface)
		}, "show onu interface xgs/1/1 status"},
		{"ONUEthernetPorts", func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error) {
			return s.ONUEthernetPorts(ctx, oltID, iface)
		}, "show onu interface xgs/1/1 eth all"},
		{"DHCPSnoopingEntries", func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error) {
			return s.DHCPSnoopingEntries(ctx, oltID, iface)
		}, "show dhcpsnooping interface xgs/1/1"},
		{"MACAddressTableEntries", func(s *KontronService, ctx context.Context, oltID uuid.UUID) (string, error) {
			return s.MACAddressTableEntries(ctx, oltID, iface)
		}, "show mac-addr-table interface xgs/1/1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oltID := uuid.New()
			shell := &fakeShell{output: "sample output"}
			dialer := &fakeDialer{shell: shell}
			s := NewKontronService(dialer)

			got, err := tc.call(s, context.Background(), oltID)
			if err != nil {
				t.Fatalf("%s() = %v", tc.name, err)
			}
			if got != "sample output" {
				t.Errorf("%s() = %q, want %q", tc.name, got, "sample output")
			}
			if shell.gotCommand != tc.wantCommand {
				t.Errorf("command sent = %q, want %q", shell.gotCommand, tc.wantCommand)
			}
			if !shell.closeCalled {
				t.Error("shell was not closed")
			}
		})
	}
}

func TestRunClosesShellEvenWhenCommandFails(t *testing.T) {
	shell := &fakeShell{err: errors.New("connection reset")}
	dialer := &fakeDialer{shell: shell}
	s := NewKontronService(dialer)

	if _, err := s.ONUSummary(context.Background(), uuid.New()); err == nil {
		t.Fatal("ONUSummary() = nil error, want the shell's failure surfaced")
	}
	if !shell.closeCalled {
		t.Error("shell was not closed after a failed command")
	}
}

func TestRunClassifiesDialFailureAsUnavailable(t *testing.T) {
	dialer := &fakeDialer{err: errors.New("dial tcp: connection refused")}
	s := NewKontronService(dialer)

	_, err := s.ONUSummary(context.Background(), uuid.New())
	if !apperror.Is(err, apperror.KindUnavailable) {
		t.Fatalf("error kind = %v, want %v", apperror.KindOf(err), apperror.KindUnavailable)
	}
}

func TestRunPropagatesAlreadyClassifiedDialError(t *testing.T) {
	conflictErr := apperror.Conflict("OLT has no connection profile configured")
	dialer := &fakeDialer{err: conflictErr}
	s := NewKontronService(dialer)

	_, err := s.ONUSummary(context.Background(), uuid.New())
	if !errors.Is(err, conflictErr) {
		t.Fatalf("error = %v, want %v returned unchanged", err, conflictErr)
	}
}

func TestRunClassifiesCommandFailureAsUnavailable(t *testing.T) {
	shell := &fakeShell{err: errors.New("ssh: interactive shell read failed: EOF")}
	dialer := &fakeDialer{shell: shell}
	s := NewKontronService(dialer)

	_, err := s.ONUSummary(context.Background(), uuid.New())
	if !apperror.Is(err, apperror.KindUnavailable) {
		t.Fatalf("error kind = %v, want %v", apperror.KindOf(err), apperror.KindUnavailable)
	}
	if !shell.closeCalled {
		t.Error("shell was not closed")
	}
}

// TestRunClassifiesInvalidInterfaceAsInvalid proves an interface value
// kontron.Client itself rejects (an embedded newline, per
// kontron.ErrInvalidInterface) is reclassified as apperror.KindInvalid
// rather than KindUnavailable — a caller input problem, not a
// connectivity one — and that the already-opened shell is still closed
// even though the command itself never reached it.
func TestRunClassifiesInvalidInterfaceAsInvalid(t *testing.T) {
	shell := &fakeShell{output: "should never be reached"}
	dialer := &fakeDialer{shell: shell}
	s := NewKontronService(dialer)

	_, err := s.ONUDetail(context.Background(), uuid.New(), "xgs/1/1\nreload")
	if !errors.Is(err, kontron.ErrInvalidInterface) {
		t.Fatalf("error = %v, want it to wrap %v", err, kontron.ErrInvalidInterface)
	}
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("error kind = %v, want %v", apperror.KindOf(err), apperror.KindInvalid)
	}
	if shell.gotCommand != "" {
		t.Errorf("shell.RunCommand was called with %q; it must never be reached for an invalid interface", shell.gotCommand)
	}
	if !shell.closeCalled {
		t.Error("shell was not closed even though it was successfully opened")
	}
}
