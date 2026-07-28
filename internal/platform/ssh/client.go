package ssh

import (
	"context"
	"errors"
	"fmt"
	"time"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// session is the subset of *golang.org/x/crypto/ssh.Session this package
// depends on. *gossh.Session already satisfies it structurally — no
// adapter is needed on that side, only on connection below, where the
// method golang.org/x/crypto/ssh itself exposes returns a concrete
// *gossh.Session rather than this interface.
type session interface {
	Output(cmd string) ([]byte, error)
	Close() error
}

// connection is the subset of *golang.org/x/crypto/ssh.Client this
// package depends on. Defining it (and session above) as interfaces,
// rather than depending on golang.org/x/crypto/ssh's concrete types
// directly, is what makes newClient testable without a live SSH server
// (goal 6): a test supplies a fake connection/session pair through
// dialFunc, and everything below this line — timeout handling, context
// cancellation, exit-status interpretation, error wrapping — runs
// exactly as it would against a real connection.
type connection interface {
	NewSession() (session, error)
	Close() error
}

// dialFunc opens a connection to addr using config. defaultDial is the
// production implementation, wrapping golang.org/x/crypto/ssh.Dial;
// client_test.go substitutes a fake dialFunc for every test, so none of
// them touch a real network.
type dialFunc func(network, addr string, config *gossh.ClientConfig) (connection, error)

// defaultDial wraps golang.org/x/crypto/ssh.Dial, adapting its
// concrete *gossh.Client return value to this package's connection
// interface via realConnection.
func defaultDial(network, addr string, config *gossh.ClientConfig) (connection, error) {
	c, err := gossh.Dial(network, addr, config)
	if err != nil {
		return nil, err
	}
	return realConnection{c}, nil
}

// realConnection adapts *gossh.Client to connection. The only reason
// this adapter exists at all is that (*gossh.Client).NewSession returns
// a concrete *gossh.Session, not this package's session interface —
// Go's structural typing does not extend to method return types needing
// an explicit conversion like this.
type realConnection struct {
	*gossh.Client
}

func (r realConnection) NewSession() (session, error) {
	s, err := r.Client.NewSession()
	if err != nil {
		return nil, err
	}
	return s, nil
}

// client is Client's one implementation.
type client struct {
	conn    connection
	timeout time.Duration
	closed  bool
}

var _ Client = (*client)(nil)

// newClient is New's actual implementation, taking dial as a parameter
// so client_test.go can supply a fake without New's own signature (fixed
// by this milestone's public API) needing to expose that seam.
func newClient(cfg Config, dial dialFunc) (*client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	var authMethods []gossh.AuthMethod
	if cfg.Password != "" {
		authMethods = append(authMethods, gossh.Password(cfg.Password))
	}
	if len(cfg.PrivateKey) > 0 {
		signer, err := gossh.ParsePrivateKey(cfg.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("ssh: parse private key: %w", err)
		}
		authMethods = append(authMethods, gossh.PublicKeys(signer))
	}

	var hostKeyCallback gossh.HostKeyCallback
	if cfg.StrictHostKeyChecking {
		callback, err := knownhosts.New(cfg.KnownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("ssh: load known hosts file %q: %w", cfg.KnownHostsFile, err)
		}
		hostKeyCallback = callback
	} else {
		// Explicitly the lab/test escape hatch this package's doc
		// comment documents — never the default (see Config.validate,
		// which does not require this branch, and Config's own doc
		// comment on StrictHostKeyChecking's zero value).
		hostKeyCallback = gossh.InsecureIgnoreHostKey()
	}

	timeout := cfg.timeout()
	clientConfig := &gossh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	conn, err := dial("tcp", cfg.addr(), clientConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh: dial %s: %w", cfg.addr(), err)
	}

	return &client{conn: conn, timeout: timeout}, nil
}

// Run implements Client.
//
// It derives a per-call deadline from c.timeout (Config.Timeout, or
// DefaultTimeout) on top of whatever ctx already carries, then races the
// remote command against that combined context: golang.org/x/crypto/ssh
// has no context-aware API of its own (Session.Output is a plain
// blocking call), so honoring cancellation means running Output in a
// goroutine and selecting between it finishing and the context ending,
// closing the session to unblock the goroutine if the context wins.
func (c *client) Run(ctx context.Context, command string) (string, error) {
	if c.closed {
		return "", ErrClientClosed
	}

	runCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	sess, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh: open session: %w", err)
	}
	defer sess.Close()

	type outcome struct {
		output []byte
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		output, err := sess.Output(command)
		done <- outcome{output, err}
	}()

	select {
	case <-runCtx.Done():
		// Best-effort: closing the session should unblock the still-
		// running goroutine's Output call. The goroutine's result is
		// discarded either way — the caller has already moved on.
		_ = sess.Close()
		return "", fmt.Errorf("ssh: run command: %w", runCtx.Err())

	case o := <-done:
		if o.err != nil {
			// A non-zero exit status (or a session torn down cleanly
			// with no exit status at all) is not this package's concern
			// — see Run's own doc comment on why interpreting it is out
			// of scope. Whatever stdout was captured before the command
			// exited is still meaningful and is returned as a success.
			var exitErr *gossh.ExitError
			var exitMissingErr *gossh.ExitMissingError
			if errors.As(o.err, &exitErr) || errors.As(o.err, &exitMissingErr) {
				return string(o.output), nil
			}
			return "", fmt.Errorf("ssh: run command: %w", o.err)
		}
		return string(o.output), nil
	}
}

// Close implements Client.
func (c *client) Close() error {
	c.closed = true
	return c.conn.Close()
}
