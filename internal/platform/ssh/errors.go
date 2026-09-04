package ssh

import "errors"

// Sentinel errors for well-known, well-named failure conditions this
// package itself detects — Config validation failures, and Run being
// called after Close. Everything else (a dial failure, an authentication
// rejection, an I/O error mid-command) comes from
// golang.org/x/crypto/ssh or the standard library and is returned
// wrapped (via fmt.Errorf's %w) rather than translated into a sentinel
// of this package's own, so errors.Is/errors.As against the underlying
// library's own error types (e.g. *ssh.ExitError, x/crypto/ssh's
// *ssh.PermissionError, net.Error for a timeout) still works for a
// caller that needs that level of detail. See this package's own doc
// comment for why that wrapping stops short of apperror translation.
var (
	// ErrHostRequired means Config.Host was empty.
	ErrHostRequired = errors.New("ssh: Host is required")

	// ErrUsernameRequired means Config.Username was empty.
	ErrUsernameRequired = errors.New("ssh: Username is required")

	// ErrNoAuthMethod means neither Config.Password nor
	// Config.PrivateKey was set. See this package's doc comment,
	// "Authentication," for why at least one is required.
	ErrNoAuthMethod = errors.New("ssh: at least one of Password or PrivateKey must be set")

	// ErrKnownHostsFileRequired means Config.StrictHostKeyChecking was
	// true but Config.KnownHostsFile was empty — there is nothing to
	// verify the server's host key against.
	ErrKnownHostsFileRequired = errors.New("ssh: KnownHostsFile is required when StrictHostKeyChecking is true")

	// ErrClientClosed means Run was called on a Client after Close.
	ErrClientClosed = errors.New("ssh: client is closed")

	// ErrShellClosed means RunCommand was called on a Shell after Close.
	ErrShellClosed = errors.New("ssh: shell is closed")

	// ErrPromptNotDetected means Client.Interactive gave up waiting for
	// something prompt-shaped (see this package's doc comment,
	// "Interactive shell mode") to appear after opening the shell
	// channel, within promptDetectionTimeout. This usually means either
	// the remote side printed something during login that never settled
	// into a recognizable prompt, or the device does not behave the way
	// this package's prompt-detection heuristic expects.
	ErrPromptNotDetected = errors.New("ssh: no command prompt detected on interactive shell")
)
