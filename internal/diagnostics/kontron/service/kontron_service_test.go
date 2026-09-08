package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// unusedOLTLister and unusedOLTModelGetter satisfy KontronService's two
// AggregatedBlacklist-only dependencies for every test in this file that
// exercises some other method — mirroring
// internal/olt/service_test.go's fakePONPortRepository panic-if-called
// pattern for a dependency a given test has no business reaching.
type unusedOLTLister struct{}

func (unusedOLTLister) List(context.Context) ([]olt.OLT, error) {
	panic("not used by this test")
}

type unusedOLTModelGetter struct{}

func (unusedOLTModelGetter) Get(context.Context, uuid.UUID) (oltmodel.OLTModel, error) {
	panic("not used by this test")
}

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
	s := NewKontronService(dialer, unusedOLTLister{}, unusedOLTModelGetter{})

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
			s := NewKontronService(dialer, unusedOLTLister{}, unusedOLTModelGetter{})

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
	s := NewKontronService(dialer, unusedOLTLister{}, unusedOLTModelGetter{})

	if _, err := s.ONUSummary(context.Background(), uuid.New()); err == nil {
		t.Fatal("ONUSummary() = nil error, want the shell's failure surfaced")
	}
	if !shell.closeCalled {
		t.Error("shell was not closed after a failed command")
	}
}

func TestRunClassifiesDialFailureAsUnavailable(t *testing.T) {
	dialer := &fakeDialer{err: errors.New("dial tcp: connection refused")}
	s := NewKontronService(dialer, unusedOLTLister{}, unusedOLTModelGetter{})

	_, err := s.ONUSummary(context.Background(), uuid.New())
	if !apperror.Is(err, apperror.KindUnavailable) {
		t.Fatalf("error kind = %v, want %v", apperror.KindOf(err), apperror.KindUnavailable)
	}
}

func TestRunPropagatesAlreadyClassifiedDialError(t *testing.T) {
	conflictErr := apperror.Conflict("OLT has no connection profile configured")
	dialer := &fakeDialer{err: conflictErr}
	s := NewKontronService(dialer, unusedOLTLister{}, unusedOLTModelGetter{})

	_, err := s.ONUSummary(context.Background(), uuid.New())
	if !errors.Is(err, conflictErr) {
		t.Fatalf("error = %v, want %v returned unchanged", err, conflictErr)
	}
}

func TestRunClassifiesCommandFailureAsUnavailable(t *testing.T) {
	shell := &fakeShell{err: errors.New("ssh: interactive shell read failed: EOF")}
	dialer := &fakeDialer{shell: shell}
	s := NewKontronService(dialer, unusedOLTLister{}, unusedOLTModelGetter{})

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
	s := NewKontronService(dialer, unusedOLTLister{}, unusedOLTModelGetter{})

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

// fakeOLTLister is an in-memory oltLister.
type fakeOLTLister struct {
	olts []olt.OLT
}

func (f *fakeOLTLister) List(context.Context) ([]olt.OLT, error) {
	return f.olts, nil
}

// fakeOLTModelGetter is an in-memory oltModelGetter, keyed by OLTModelID.
type fakeOLTModelGetter struct {
	byID map[uuid.UUID]oltmodel.OLTModel
}

func (f *fakeOLTModelGetter) Get(_ context.Context, id uuid.UUID) (oltmodel.OLTModel, error) {
	model, ok := f.byID[id]
	if !ok {
		return oltmodel.OLTModel{}, apperror.NotFound("olt model not found")
	}
	return model, nil
}

// multiDialer is a dialer keyed by OLT ID, so AggregatedBlacklist's
// tests can give different (or failing) OLTs different responses in one
// fan-out — unlike fakeDialer above, which every other test in this file
// only ever points at one OLT at a time.
type multiDialer struct {
	byOLTID map[uuid.UUID]struct {
		shell ssh.Shell
		err   error
	}
}

func (d *multiDialer) Dial(_ context.Context, oltID uuid.UUID) (ssh.Shell, error) {
	entry, ok := d.byOLTID[oltID]
	if !ok {
		panic("dialed an OLT this test never configured")
	}
	if entry.err != nil {
		return nil, entry.err
	}
	return entry.shell, nil
}

// TestAggregatedBlacklistMergesAcrossReachableKontronOLTsAndSkipsOthers
// covers every branch AggregatedBlacklist's own doc comment describes:
// a reachable Kontron OLT contributes its entries, an unreachable
// Kontron OLT is reported separately instead of failing the whole call,
// and a non-Kontron OLT is never dialed at all.
func TestAggregatedBlacklistMergesAcrossReachableKontronOLTsAndSkipsOthers(t *testing.T) {
	kontronModelID, otherModelID := uuid.New(), uuid.New()
	reachableID, unreachableID, otherVendorID := uuid.New(), uuid.New(), uuid.New()

	olts := []olt.OLT{
		{ID: reachableID, Name: "reachable-olt", OLTModelID: kontronModelID},
		{ID: unreachableID, Name: "unreachable-olt", OLTModelID: kontronModelID},
		{ID: otherVendorID, Name: "other-vendor-olt", OLTModelID: otherModelID},
	}
	models := &fakeOLTModelGetter{byID: map[uuid.UUID]oltmodel.OLTModel{
		kontronModelID: {ID: kontronModelID, Vendor: oltmodel.VendorKontron, Name: "C16", PONPortCount: 16},
		otherModelID:   {ID: otherModelID, Vendor: oltmodel.VendorNokia, Name: "Some Nokia Chassis", PONPortCount: 8},
	}}

	reachableShell := &fakeShell{output: realBlacklistSampleForAggregationTest}
	dial := &multiDialer{byOLTID: map[uuid.UUID]struct {
		shell ssh.Shell
		err   error
	}{
		reachableID:   {shell: reachableShell},
		unreachableID: {err: errors.New("dial tcp: connection refused")},
		// otherVendorID deliberately has no entry: multiDialer.Dial
		// panics if it is ever looked up, proving the non-Kontron OLT
		// was never dialed.
	}}

	s := NewKontronService(dial, &fakeOLTLister{olts: olts}, models)

	result, err := s.AggregatedBlacklist(context.Background())
	if err != nil {
		t.Fatalf("AggregatedBlacklist() = %v", err)
	}

	if len(result.ONUs) != 1 {
		t.Fatalf("len(ONUs) = %d, want 1: %+v", len(result.ONUs), result.ONUs)
	}
	got := result.ONUs[0]
	want := BlacklistedONU{
		OLTID:          reachableID,
		OLTName:        "reachable-olt",
		Interface:      "xgs/6",
		SerialNumber:   "ISKT2308DD88",
		RegistrationID: `""`,
		Cause:          "Serial Number not known",
	}
	if got != want {
		t.Errorf("ONUs[0] = %+v, want %+v", got, want)
	}

	if len(result.Unreachable) != 1 {
		t.Fatalf("len(Unreachable) = %d, want 1: %+v", len(result.Unreachable), result.Unreachable)
	}
	if result.Unreachable[0].OLTID != unreachableID {
		t.Errorf("Unreachable[0].OLTID = %v, want %v", result.Unreachable[0].OLTID, unreachableID)
	}
	if result.Unreachable[0].Reason == "" {
		t.Error("Unreachable[0].Reason is empty, want the dial failure's message")
	}
}

// realBlacklistSampleForAggregationTest is the same real captured sample
// internal/diagnostics/kontron's own blacklist_test.go asserts against.
const realBlacklistSampleForAggregationTest = "jamestown-allen-olt-02#show onu black-list\n" +
	"Interface  Serial Number     Password/Registration Id                Cause\n" +
	"---------  ----------------  --------------------------------------  -----------------------\n" +
	"xgs/6      ISKT2308DD88      \"\"                                      Serial Number not known\n" +
	"---------------------------\n" +
	"Total: 1\n"

// TestAggregatedBlacklistFailsWhenAnOLTModelCannotBeLoaded proves the
// distinction AggregatedBlacklist's own doc comment draws: a broken
// OLTModel lookup (a data-integrity problem the foreign key should have
// prevented) fails the whole call, unlike an individual OLT being
// unreachable.
func TestAggregatedBlacklistFailsWhenAnOLTModelCannotBeLoaded(t *testing.T) {
	oltID := uuid.New()
	olts := []olt.OLT{{ID: oltID, Name: "olt-with-missing-model", OLTModelID: uuid.New()}}
	models := &fakeOLTModelGetter{byID: map[uuid.UUID]oltmodel.OLTModel{}}
	dial := &multiDialer{byOLTID: map[uuid.UUID]struct {
		shell ssh.Shell
		err   error
	}{}}

	s := NewKontronService(dial, &fakeOLTLister{olts: olts}, models)

	if _, err := s.AggregatedBlacklist(context.Background()); err == nil {
		t.Fatal("AggregatedBlacklist() = nil error, want the OLTModel lookup failure surfaced")
	}
}
