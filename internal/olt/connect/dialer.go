package connect

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// oltGetter, connectionProfileGetter, and authenticationGetter are the
// seams Dialer (and NewDialer itself) depend on instead of the three
// full repository interfaces — mirroring the narrowing pattern
// internal/workflow/engine.NewDefaultEngine's own transitioner parameter
// already establishes in this codebase: Dial only ever fetches one
// record of each kind, by ID, so that is all it declares a dependency
// on. Every concrete repository (e.g. a real olt.OLTRepository
// implementation) already satisfies its narrower counterpart here
// structurally, so nothing is lost at a real call site — only
// dialer_test.go's fakes, which implement Get alone, benefit from not
// also having to stub List/Create/Update/Delete.
type oltGetter interface {
	Get(ctx context.Context, id uuid.UUID) (olt.OLT, error)
}

type connectionProfileGetter interface {
	Get(ctx context.Context, id uuid.UUID) (connectionprofile.ConnectionProfile, error)
}

type authenticationGetter interface {
	Get(ctx context.Context, id uuid.UUID) (authentication.Authentication, error)
}

// dialShellFunc matches Shell's own signature, extracted so Dialer's
// tests can substitute a fake that never touches the network — the same
// reasoning internal/platform/ssh.New's own dialFunc parameter exists
// for newClient.
type dialShellFunc func(ctx context.Context, target olt.OLT, profile connectionprofile.ConnectionProfile, auth authentication.Authentication, knownHostsFile string) (ssh.Shell, error)

// Dialer resolves an OLT by ID — along with the ConnectionProfile and
// Authentication it references — and opens a live ssh.Shell to it. It is
// the ID-based counterpart to Shell: where Shell takes already-loaded
// olt.OLT / connectionprofile.ConnectionProfile /
// authentication.Authentication values, Dialer.Dial takes only an OLT's
// UUID and performs the repository lookups itself, in order, failing
// fast the same way internal/workflow/engine.DefaultEngine.Execute does.
type Dialer struct {
	olts               oltGetter
	connectionProfiles connectionProfileGetter
	authentications    authenticationGetter
	knownHostsFile     string
	dialShell          dialShellFunc
}

// NewDialer builds a Dialer. Real callers pass their full
// olt.OLTRepository / connectionprofile.ConnectionProfileRepository /
// authentication.AuthenticationRepository implementations directly —
// each already satisfies the narrower interface NewDialer actually
// declares, per this file's own doc comment on oltGetter. knownHostsFile
// is passed straight through to Shell on every call — see that
// function's own doc comment on when it is actually used.
func NewDialer(
	olts oltGetter,
	connectionProfiles connectionProfileGetter,
	authentications authenticationGetter,
	knownHostsFile string,
) *Dialer {
	return newDialer(olts, connectionProfiles, authentications, knownHostsFile, Shell)
}

// newDialer is NewDialer's actual implementation, taking dialShell as a
// parameter so dialer_test.go can supply a fake without NewDialer's own
// signature (the public API) needing to expose that seam — the same
// shape internal/platform/ssh.New/newClient already establishes.
func newDialer(
	olts oltGetter,
	connectionProfiles connectionProfileGetter,
	authentications authenticationGetter,
	knownHostsFile string,
	dialShell dialShellFunc,
) *Dialer {
	return &Dialer{
		olts:               olts,
		connectionProfiles: connectionProfiles,
		authentications:    authentications,
		knownHostsFile:     knownHostsFile,
		dialShell:          dialShell,
	}
}

// Dial resolves the OLT identified by oltID, its ConnectionProfile, and
// that profile's Authentication, then opens an ssh.Shell to it via
// Shell.
//
// A not-found error from any of the three Get calls is returned exactly
// as that repository produced it (already an apperror.KindNotFound
// error, by this codebase's established convention). An OLT with no
// ConnectionProfile bound, or a ConnectionProfile with no Authentication
// bound, is a different kind of failure — the referenced records
// genuinely do not exist yet, not that a lookup failed — reported as
// apperror.KindConflict: the OLT (or profile) exists, but its current
// state does not yet allow a connection to be made.
func (d *Dialer) Dial(ctx context.Context, oltID uuid.UUID) (ssh.Shell, error) {
	o, err := d.olts.Get(ctx, oltID)
	if err != nil {
		return nil, err
	}

	if o.ConnectionProfileID == nil {
		return nil, apperror.Conflict(fmt.Sprintf("olt connect: OLT %s has no connection profile configured", o.ID))
	}

	profile, err := d.connectionProfiles.Get(ctx, *o.ConnectionProfileID)
	if err != nil {
		return nil, err
	}

	if profile.AuthenticationID == nil {
		return nil, apperror.Conflict(fmt.Sprintf("olt connect: connection profile %s has no authentication configured", profile.ID))
	}

	auth, err := d.authentications.Get(ctx, *profile.AuthenticationID)
	if err != nil {
		return nil, err
	}

	return d.dialShell(ctx, o, profile, auth, d.knownHostsFile)
}
