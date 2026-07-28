// Package ssh is Palladium's reusable SSH platform package: a thin,
// general-purpose wrapper around golang.org/x/crypto/ssh for opening a
// connection to a remote host, running exactly one command, and
// returning its output. It is infrastructure, not a feature.
//
// This package is explicitly NOT any of the following:
//
//   - Kontron support, or support for any other OLT/router/CPE vendor.
//     Per CLAUDE.md's Plugin Philosophy ("The core system must never
//     contain Kontron-, Nokia-, Calix-, Adtran-, MikroTik-, or
//     vendor-specific logic"), this package knows nothing about what
//     command a Kontron OLT expects or how to interpret its output —
//     that belongs entirely to a future vendor plugin built on top of
//     this one.
//   - Diagnostics (see internal/diagnostics). A future Diagnostic
//     implementation will use a Client to reach a device; this package
//     has no notion of an ONU, a Result, or a Section.
//   - Provisioning (see internal/provisioning). A future Connector
//     implementation will likewise use a Client; this package has no
//     notion of a Service, a ProvisioningJob, or an operation.
//
// Because of the above, this package has zero dependency on any
// business domain in this codebase — only the standard library and
// golang.org/x/crypto/ssh (already a dependency of internal/auth's
// password hashing). It does not import internal/platform/apperror
// either, for the same reason internal/provisioning/connectors doesn't
// (see that package's doc comment on Connector's error type): this
// package models an operational boundary — talking to an external
// system over the network — not an HTTP-facing API boundary, so it has
// no reason to carry apperror's Kind taxonomy. A future caller that does
// sit closer to that boundary (a diagnostics execution path, a
// provisioning connector) is responsible for deciding how a Client
// error should ultimately be reported to a user.
//
// # Lifecycle
//
// A Client represents one open SSH connection. Call New to open it,
// call Run as many times as needed — but never concurrently; see Run's
// own doc comment — and call Close exactly once when done, typically via
// defer immediately after a successful New. Run after Close returns
// ErrClientClosed rather than panicking or blocking.
//
// # Authentication
//
// Config.Password and Config.PrivateKey are both optional individually,
// but New rejects a Config with neither set (see ErrNoAuthMethod): a
// Client with no way to authenticate can never be useful. Both may be
// set together, in which case both are offered to the server as
// candidate authentication methods, in the order golang.org/x/crypto/ssh
// itself tries them. PrivateKey is the raw PEM-encoded key bytes (e.g.
// the contents of an id_ed25519 file read via os.ReadFile), parsed via
// ssh.ParsePrivateKey — an encrypted (passphrase-protected) private key
// is not supported, since nothing in this milestone's scope calls for
// it.
//
// # Timeout behavior
//
// Config.Timeout serves two distinct purposes, both governed by the
// same value: it bounds how long New waits for the initial TCP
// connection and SSH handshake to complete (passed straight through to
// golang.org/x/crypto/ssh.ClientConfig.Timeout), and it bounds each
// individual Run call by deriving a context.WithTimeout from the ctx the
// caller passed in. A zero Config.Timeout is not "no timeout" — it
// defaults to DefaultTimeout (see that constant) — because a Client
// with no bound on either operation could hang indefinitely on a
// misbehaving or unreachable host, and nothing about this package's
// intended use (diagnostics, provisioning — both explicitly latency-
// sensitive, request/response operational flows) benefits from that.
// Run also honors ctx's own cancellation and any deadline it already
// carries, independent of Config.Timeout — whichever fires first wins.
//
// # Host-key verification
//
// Config.StrictHostKeyChecking controls whether the server's host key
// is verified against Config.KnownHostsFile (via
// golang.org/x/crypto/ssh/knownhosts) or accepted unconditionally via
// ssh.InsecureIgnoreHostKey.
//
// StrictHostKeyChecking is a plain bool, and Go's zero value for bool is
// false — meaning a zero-value Config skips host-key verification
// entirely. This is a deliberate, security-relevant consequence of using
// the exact field name and type this milestone specifies, not an
// oversight: every caller constructing a Config must explicitly write
// StrictHostKeyChecking: true to get verification, the same way every
// other security-relevant default in this codebase requires an explicit
// opt-in rather than an accidental opt-out. Production call sites (a
// future Kontron plugin, for instance) should always set it to true and
// supply a real KnownHostsFile; StrictHostKeyChecking: false exists
// specifically for the lab/test scenario this milestone names explicitly
// ("Support disabling verification explicitly for labs").
//
// # Intended future use
//
// This package exists to be imported by two future milestones named
// explicitly in its own scope: a diagnostics implementation that needs
// to actually reach an ONU or OLT (see internal/diagnostics's current
// placeholder, BasicONUCheck, which this package will eventually back),
// and a provisioning connector (see internal/provisioning/connectors)
// that needs to configure a device over SSH. Neither exists yet; this
// package is deliberately usable by both without knowing anything about
// either.
package ssh

import (
	"context"
	"time"
)

// DefaultTimeout is used for both connection establishment and each Run
// call when Config.Timeout is zero. See this package's doc comment,
// "Timeout behavior," for why a zero Config.Timeout does not mean "no
// timeout."
const DefaultTimeout = 30 * time.Second

// Client is an open SSH connection capable of running commands on the
// remote host, one at a time. See this package's doc comment for its
// full lifecycle, authentication, timeout, and host-key verification
// behavior.
type Client interface {
	// Run executes command on the remote host and returns its standard
	// output.
	//
	// Run is not safe for concurrent use: per this milestone's explicit
	// scope, concurrent command execution is out of scope, and a Client
	// makes no attempt to serialize overlapping Run calls itself — a
	// caller needing to run several commands concurrently must use
	// separate Client instances (separate New calls), not share one
	// across goroutines.
	//
	// The remote command's own exit status is deliberately not treated
	// as an error: interpreting what a given command's exit code means
	// is inherently command- and vendor-specific (this milestone's
	// explicit "no command parsing" boundary), so Run returns whatever
	// standard output was captured regardless of exit status, and
	// reserves a non-nil error strictly for failures of the SSH
	// mechanism itself — the session could not be opened, the
	// connection was lost mid-command, ctx was cancelled or its
	// deadline (or Config.Timeout's derived one) was exceeded, or the
	// Client has already been closed.
	Run(ctx context.Context, command string) (string, error)

	// Close closes the underlying SSH connection. It is safe to call
	// Close even if Run was never called. Calling Close more than once
	// returns the error from the underlying connection's own repeated
	// close (typically nil, since closing an already-closed connection
	// is usually a no-op) — Close does not itself guard against being
	// called twice.
	Close() error
}

// New opens a Client to the host and port described by cfg,
// authenticating with whichever of Config.Password and
// Config.PrivateKey are set. It returns an error without ever opening a
// network connection if cfg is invalid — see Config's own doc comment
// for what "invalid" means — and otherwise blocks until the connection
// and SSH handshake complete or Config.Timeout (or DefaultTimeout, if
// Config.Timeout is zero) elapses.
func New(cfg Config) (Client, error) {
	c, err := newClient(cfg, defaultDial)
	if err != nil {
		// Deliberately not `return newClient(cfg, defaultDial)` directly:
		// newClient returns a concrete *client, and a nil *client
		// converted into the Client interface is a non-nil interface
		// value (the classic Go "typed nil" trap) — a caller's `client
		// != nil` check would then see a non-nil Client even though
		// nothing usable was ever created. Returning a literal nil here
		// avoids that entirely.
		return nil, err
	}
	return c, nil
}
