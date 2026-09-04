// Package connect resolves an already-loaded OLT, ConnectionProfile, and
// Authentication into a live, ready-to-use internal/platform/ssh.Shell.
// It is the piece named as "a future caller" throughout
// internal/connectionprofile and internal/authentication's own doc
// comments ("nothing in this package opens a connection") — this is that
// caller, for the OLT case specifically.
//
// This package deliberately does not look anything up by ID: it takes
// olt.OLT, connectionprofile.ConnectionProfile, and
// authentication.Authentication as already-resolved values, not UUIDs.
// Chasing OLT.ConnectionProfileID -> ConnectionProfile.AuthenticationID
// -> Authentication through their respective repositories is a separate
// concern, left to whatever future service actually triggers a
// diagnostic or provisioning operation against a specific OLT — building
// that orchestration now, with no caller yet to shape it, would be
// guessing.
//
// It is also deliberately Kontron-agnostic, unlike
// internal/diagnostics/kontron: nothing here knows or cares which
// vendor's OLT it is connecting to (olt.OLT.Vendor is never inspected) —
// a future Nokia or Calix integration would resolve a connection through
// this exact same package.
package connect

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// ErrManagementAddressRequired means target.ManagementIPAddress was
// empty — there is nowhere to dial.
var ErrManagementAddressRequired = errors.New("olt connect: OLT has no management IP address")

// Shell opens an interactive SSH shell to target, authenticating and
// configuring the connection according to profile and auth (see
// buildConfig for exactly how their fields map to an ssh.Config).
//
// It always uses ssh.Client.Interactive, never ssh.Client.Run: every OLT
// confirmed against so far (a Kontron/Iskratel C16) requires the
// interactive shell path — see internal/platform/ssh's own "Interactive
// shell mode" doc comment — and nothing in this milestone's scope calls
// for a device that would need the plain exec channel instead.
//
// knownHostsFile is used only when profile.HostKeyPolicy is
// connectionprofile.HostKeyPolicyStrict (see ssh.Config's own doc
// comment on StrictHostKeyChecking); it is ignored otherwise, and may be
// empty when nothing in the caller's configuration requires strict
// checking. This package does not read it from anywhere itself — no
// global configuration, no default path — the caller is expected to
// supply it explicitly, the same way it already supplies target,
// profile, and auth.
//
// The returned Shell's Close also closes the underlying ssh.Client, so a
// caller has exactly one thing to close — see this package's own
// unexported dialedShell for why that matters concretely for a device
// with a small concurrent-session budget like the C16's.
func Shell(ctx context.Context, target olt.OLT, profile connectionprofile.ConnectionProfile, auth authentication.Authentication, knownHostsFile string) (ssh.Shell, error) {
	cfg, err := buildConfig(target, profile, auth, knownHostsFile)
	if err != nil {
		return nil, err
	}

	client, err := ssh.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("olt connect: %w", err)
	}

	sh, err := client.Interactive(ctx)
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("olt connect: %w", err)
	}

	return &dialedShell{Shell: sh, client: client}, nil
}

// buildConfig translates target, profile, and auth into an ssh.Config,
// with no network I/O of its own — see Shell, its one caller.
func buildConfig(target olt.OLT, profile connectionprofile.ConnectionProfile, auth authentication.Authentication, knownHostsFile string) (ssh.Config, error) {
	if target.ManagementIPAddress == "" {
		return ssh.Config{}, ErrManagementAddressRequired
	}

	// Protocol is a plain, unvalidated string on ConnectionProfile (see
	// that package's own validate.go doc comment on why) — empty is a
	// legitimate "no protocol chosen yet" template value, treated the
	// same as "ssh" since this package speaks nothing else; anything
	// else explicit is rejected rather than silently attempted over SSH
	// anyway.
	if profile.Protocol != "" && !strings.EqualFold(profile.Protocol, "ssh") {
		return ssh.Config{}, fmt.Errorf("olt connect: unsupported connection profile protocol %q (only ssh is supported)", profile.Protocol)
	}

	cfg := ssh.Config{
		Host:     target.ManagementIPAddress,
		Port:     profile.Port,
		Username: auth.Username,
		Timeout:  profile.Timeout,
	}

	switch auth.AuthenticationType {
	case authentication.AuthenticationTypePassword:
		cfg.Password = auth.Password
	case authentication.AuthenticationTypeSSHKey:
		cfg.PrivateKey = []byte(auth.PrivateKey)
	default:
		return ssh.Config{}, fmt.Errorf("olt connect: unsupported authentication type %q", auth.AuthenticationType)
	}

	switch profile.HostKeyPolicy {
	case connectionprofile.HostKeyPolicyStrict:
		cfg.StrictHostKeyChecking = true
		cfg.KnownHostsFile = knownHostsFile
	case connectionprofile.HostKeyPolicyInsecure:
		// StrictHostKeyChecking already defaults to false.
	default:
		return ssh.Config{}, fmt.Errorf("olt connect: unsupported host key policy %q", profile.HostKeyPolicy)
	}

	return cfg, nil
}

// dialedShell pairs an ssh.Shell with the ssh.Client it was opened from,
// so Shell's caller has exactly one thing to close. This is not just
// convenience: closing only a Shell's own channel (see ssh.Shell.Close's
// own doc comment) leaves the underlying SSH connection itself open —
// on a device like the C16, confirmed firsthand to allow as few as
// single digits of concurrent SSH/Telnet sessions in total, a caller
// that forgot to separately close the Client would leak one of those
// sessions for as long as the connection's own idle timeout allows.
type dialedShell struct {
	ssh.Shell
	client ssh.Client
}

var _ ssh.Shell = (*dialedShell)(nil)

// Close closes both the shell channel and the underlying client,
// returning the first non-nil error encountered but always attempting
// both regardless of whether the first one failed.
func (d *dialedShell) Close() error {
	shellErr := d.Shell.Close()
	clientErr := d.client.Close()
	if shellErr != nil {
		return shellErr
	}
	return clientErr
}
