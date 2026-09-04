package connect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// fakeOLTGetter, fakeConnectionProfileGetter, and
// fakeAuthenticationGetter are in-memory oltGetter /
// connectionProfileGetter / authenticationGetter implementations —
// Dialer's tests exercise the exact order and short-circuiting of its
// three lookups without a real repository or database.
type fakeOLTGetter struct {
	olt    olt.OLT
	err    error
	called bool
	gotID  uuid.UUID
}

func (f *fakeOLTGetter) Get(_ context.Context, id uuid.UUID) (olt.OLT, error) {
	f.called = true
	f.gotID = id
	if f.err != nil {
		return olt.OLT{}, f.err
	}
	return f.olt, nil
}

type fakeConnectionProfileGetter struct {
	profile connectionprofile.ConnectionProfile
	err     error
	called  bool
	gotID   uuid.UUID
}

func (f *fakeConnectionProfileGetter) Get(_ context.Context, id uuid.UUID) (connectionprofile.ConnectionProfile, error) {
	f.called = true
	f.gotID = id
	if f.err != nil {
		return connectionprofile.ConnectionProfile{}, f.err
	}
	return f.profile, nil
}

type fakeAuthenticationGetter struct {
	auth   authentication.Authentication
	err    error
	called bool
	gotID  uuid.UUID
}

func (f *fakeAuthenticationGetter) Get(_ context.Context, id uuid.UUID) (authentication.Authentication, error) {
	f.called = true
	f.gotID = id
	if f.err != nil {
		return authentication.Authentication{}, f.err
	}
	return f.auth, nil
}

// dialShellCall records one invocation of a fake dialShellFunc, for
// tests that need to assert exactly what Dial resolved and passed
// through to it.
type dialShellCall struct {
	target         olt.OLT
	profile        connectionprofile.ConnectionProfile
	auth           authentication.Authentication
	knownHostsFile string
}

func fakeDialShell(shell ssh.Shell, err error) (dialShellFunc, *[]dialShellCall) {
	calls := &[]dialShellCall{}
	fn := func(_ context.Context, target olt.OLT, profile connectionprofile.ConnectionProfile, auth authentication.Authentication, knownHostsFile string) (ssh.Shell, error) {
		*calls = append(*calls, dialShellCall{target, profile, auth, knownHostsFile})
		if err != nil {
			return nil, err
		}
		return shell, nil
	}
	return fn, calls
}

// resolvedChain builds a consistent OLT -> ConnectionProfile ->
// Authentication chain (every ID cross-referenced correctly) for tests
// that need the happy path.
func resolvedChain() (olt.OLT, connectionprofile.ConnectionProfile, authentication.Authentication) {
	profileID := uuid.New()
	authID := uuid.New()

	o := olt.OLT{ID: uuid.New(), ManagementIPAddress: "192.0.2.10", ConnectionProfileID: &profileID}
	profile := connectionprofile.ConnectionProfile{
		ID:               profileID,
		Protocol:         "ssh",
		Port:             22,
		Timeout:          5 * time.Second,
		HostKeyPolicy:    connectionprofile.HostKeyPolicyInsecure,
		AuthenticationID: &authID,
	}
	auth := authentication.Authentication{
		ID:                 authID,
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "admin",
		Password:           "secret",
	}
	return o, profile, auth
}

func TestDialResolvesFullChainAndCallsDialShell(t *testing.T) {
	o, profile, auth := resolvedChain()
	oltGetter := &fakeOLTGetter{olt: o}
	profileGetter := &fakeConnectionProfileGetter{profile: profile}
	authGetter := &fakeAuthenticationGetter{auth: auth}
	dialShell, calls := fakeDialShell(&fakeShell{}, nil)

	d := newDialer(oltGetter, profileGetter, authGetter, "/etc/palladium/known_hosts", dialShell)

	if _, err := d.Dial(context.Background(), o.ID); err != nil {
		t.Fatalf("Dial() = %v", err)
	}

	if oltGetter.gotID != o.ID {
		t.Errorf("olts.Get called with %v, want %v", oltGetter.gotID, o.ID)
	}
	if profileGetter.gotID != *o.ConnectionProfileID {
		t.Errorf("connectionProfiles.Get called with %v, want %v", profileGetter.gotID, *o.ConnectionProfileID)
	}
	if authGetter.gotID != *profile.AuthenticationID {
		t.Errorf("authentications.Get called with %v, want %v", authGetter.gotID, *profile.AuthenticationID)
	}

	if len(*calls) != 1 {
		t.Fatalf("dialShell called %d times, want 1", len(*calls))
	}
	got := (*calls)[0]
	if got.target.ID != o.ID || got.profile.ID != profile.ID || got.auth.ID != auth.ID {
		t.Errorf("dialShell called with %+v, want the resolved chain", got)
	}
	if got.knownHostsFile != "/etc/palladium/known_hosts" {
		t.Errorf("dialShell knownHostsFile = %q, want %q", got.knownHostsFile, "/etc/palladium/known_hosts")
	}
}

func TestDialPropagatesOLTNotFound(t *testing.T) {
	notFoundErr := apperror.NotFound("olt not found")
	oltGetter := &fakeOLTGetter{err: notFoundErr}
	profileGetter := &fakeConnectionProfileGetter{}
	authGetter := &fakeAuthenticationGetter{}
	dialShell, calls := fakeDialShell(&fakeShell{}, nil)

	d := newDialer(oltGetter, profileGetter, authGetter, "", dialShell)

	_, err := d.Dial(context.Background(), uuid.New())
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("Dial() error = %v, want %v", err, notFoundErr)
	}
	if profileGetter.called {
		t.Error("connectionProfiles.Get was called; it must never be reached when the OLT lookup fails")
	}
	if len(*calls) != 0 {
		t.Error("dialShell was called; it must never be reached when the OLT lookup fails")
	}
}

func TestDialReturnsConflictWhenOLTHasNoConnectionProfile(t *testing.T) {
	oltGetter := &fakeOLTGetter{olt: olt.OLT{ID: uuid.New(), ConnectionProfileID: nil}}
	profileGetter := &fakeConnectionProfileGetter{}
	authGetter := &fakeAuthenticationGetter{}
	dialShell, calls := fakeDialShell(&fakeShell{}, nil)

	d := newDialer(oltGetter, profileGetter, authGetter, "", dialShell)

	_, err := d.Dial(context.Background(), uuid.New())
	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Dial() error kind = %v, want %v", apperror.KindOf(err), apperror.KindConflict)
	}
	if profileGetter.called {
		t.Error("connectionProfiles.Get was called for an OLT with no ConnectionProfileID")
	}
	if len(*calls) != 0 {
		t.Error("dialShell was called for an OLT with no ConnectionProfileID")
	}
}

func TestDialPropagatesConnectionProfileNotFound(t *testing.T) {
	profileID := uuid.New()
	notFoundErr := apperror.NotFound("connection profile not found")
	oltGetter := &fakeOLTGetter{olt: olt.OLT{ID: uuid.New(), ConnectionProfileID: &profileID}}
	profileGetter := &fakeConnectionProfileGetter{err: notFoundErr}
	authGetter := &fakeAuthenticationGetter{}
	dialShell, calls := fakeDialShell(&fakeShell{}, nil)

	d := newDialer(oltGetter, profileGetter, authGetter, "", dialShell)

	_, err := d.Dial(context.Background(), uuid.New())
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("Dial() error = %v, want %v", err, notFoundErr)
	}
	if authGetter.called {
		t.Error("authentications.Get was called; it must never be reached when the connection profile lookup fails")
	}
	if len(*calls) != 0 {
		t.Error("dialShell was called; it must never be reached when the connection profile lookup fails")
	}
}

func TestDialReturnsConflictWhenConnectionProfileHasNoAuthentication(t *testing.T) {
	profileID := uuid.New()
	oltGetter := &fakeOLTGetter{olt: olt.OLT{ID: uuid.New(), ConnectionProfileID: &profileID}}
	profileGetter := &fakeConnectionProfileGetter{profile: connectionprofile.ConnectionProfile{ID: profileID, AuthenticationID: nil}}
	authGetter := &fakeAuthenticationGetter{}
	dialShell, calls := fakeDialShell(&fakeShell{}, nil)

	d := newDialer(oltGetter, profileGetter, authGetter, "", dialShell)

	_, err := d.Dial(context.Background(), uuid.New())
	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Dial() error kind = %v, want %v", apperror.KindOf(err), apperror.KindConflict)
	}
	if authGetter.called {
		t.Error("authentications.Get was called for a connection profile with no AuthenticationID")
	}
	if len(*calls) != 0 {
		t.Error("dialShell was called for a connection profile with no AuthenticationID")
	}
}

func TestDialPropagatesAuthenticationNotFound(t *testing.T) {
	profileID := uuid.New()
	authID := uuid.New()
	notFoundErr := apperror.NotFound("authentication not found")
	oltGetter := &fakeOLTGetter{olt: olt.OLT{ID: uuid.New(), ConnectionProfileID: &profileID}}
	profileGetter := &fakeConnectionProfileGetter{profile: connectionprofile.ConnectionProfile{ID: profileID, AuthenticationID: &authID}}
	authGetter := &fakeAuthenticationGetter{err: notFoundErr}
	dialShell, calls := fakeDialShell(&fakeShell{}, nil)

	d := newDialer(oltGetter, profileGetter, authGetter, "", dialShell)

	_, err := d.Dial(context.Background(), uuid.New())
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("Dial() error = %v, want %v", err, notFoundErr)
	}
	if len(*calls) != 0 {
		t.Error("dialShell was called; it must never be reached when the authentication lookup fails")
	}
}

func TestDialPropagatesDialShellError(t *testing.T) {
	o, profile, auth := resolvedChain()
	oltGetter := &fakeOLTGetter{olt: o}
	profileGetter := &fakeConnectionProfileGetter{profile: profile}
	authGetter := &fakeAuthenticationGetter{auth: auth}
	dialErr := errors.New("dial tcp: connection refused")
	dialShell, _ := fakeDialShell(nil, dialErr)

	d := newDialer(oltGetter, profileGetter, authGetter, "", dialShell)

	_, err := d.Dial(context.Background(), o.ID)
	if !errors.Is(err, dialErr) {
		t.Fatalf("Dial() error = %v, want %v", err, dialErr)
	}
}

// TestNewDialerWiresRealShellFunc proves NewDialer really does wire in
// the package's own Shell function (not just a fake) -- an unroutable
// management address surfaces as a genuine dial failure end to end,
// mirroring TestShellWrapsDialFailure in connect_test.go.
func TestNewDialerWiresRealShellFunc(t *testing.T) {
	o, profile, auth := resolvedChain()
	profile.Timeout = 200 * time.Millisecond
	oltGetter := &fakeOLTGetter{olt: o}
	profileGetter := &fakeConnectionProfileGetter{profile: profile}
	authGetter := &fakeAuthenticationGetter{auth: auth}

	d := NewDialer(oltGetter, profileGetter, authGetter, "")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := d.Dial(ctx, o.ID); err == nil {
		t.Fatal("Dial() = nil error, want a dial failure for an unroutable address")
	}
}
