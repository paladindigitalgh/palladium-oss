package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// generateTestPrivateKeyPEM generates a throwaway ed25519 key pair and
// PKCS8-PEM-encodes the private half — enough for
// golang.org/x/crypto/ssh.ParsePrivateKey to accept as a valid
// Config.PrivateKey in tests, without a hand-written PEM literal (fragile
// and easy to get subtly wrong) or a live SSH server. Never used to
// authenticate against anything real.
func generateTestPrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("test setup: generate ed25519 key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("test setup: marshal PKCS8 private key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// fakeSession is an in-memory session — see this file's package comment
// at the top of client.go for why connection/session exist as
// interfaces at all: this fake is what lets every test in this file run
// without a live SSH server (goal 6).
type fakeSession struct {
	output    []byte
	err       error
	closeErr  error
	closed    bool
	block     chan struct{} // if non-nil, Output blocks until this is closed
	outputted chan struct{} // closed once Output has been called, for tests that need to know the goroutine has started
}

func (s *fakeSession) Output(string) ([]byte, error) {
	if s.outputted != nil {
		close(s.outputted)
	}
	if s.block != nil {
		<-s.block
	}
	return s.output, s.err
}

func (s *fakeSession) Close() error {
	s.closed = true
	return s.closeErr
}

// fakeConnection is an in-memory connection backing fakeSession above.
type fakeConnection struct {
	session       *fakeSession
	newSessionErr error
	closeErr      error
	closed        bool

	// interactive and newInteractiveSessionErr back NewInteractiveSession,
	// exercised by interactive_test.go rather than this file — this file's
	// own tests never set them, so NewInteractiveSession returning a nil
	// interactiveSession alongside a nil error is never reached by them.
	interactive              interactiveSession
	newInteractiveSessionErr error
}

func (c *fakeConnection) NewSession() (session, error) {
	if c.newSessionErr != nil {
		return nil, c.newSessionErr
	}
	return c.session, nil
}

func (c *fakeConnection) NewInteractiveSession() (interactiveSession, error) {
	if c.newInteractiveSessionErr != nil {
		return nil, c.newInteractiveSessionErr
	}
	return c.interactive, nil
}

func (c *fakeConnection) Close() error {
	c.closed = true
	return c.closeErr
}

func validConfig() Config {
	return Config{
		Host:     "device.example.test",
		Username: "admin",
		Password: "secret",
	}
}

// fakeDial returns a dialFunc that always returns conn, ignoring
// whatever network/addr/config it was called with, and records that it
// was called at all (via called) — used by tests that need to prove a
// dial attempt was, or was not, made.
func fakeDial(conn connection, err error) (dialFunc, *bool) {
	called := false
	return func(string, string, *gossh.ClientConfig) (connection, error) {
		called = true
		if err != nil {
			return nil, err
		}
		return conn, nil
	}, &called
}

// poisonDial fails the test outright if it is ever invoked — used to
// prove Config validation short-circuits before any dial attempt.
func poisonDial(t *testing.T) dialFunc {
	t.Helper()
	return func(network, addr string, config *gossh.ClientConfig) (connection, error) {
		t.Fatalf("dial was called with an invalid Config; it must never be reached")
		return nil, nil
	}
}

func TestNewClientValidatesConfigBeforeDialing(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{"missing host", Config{}, ErrHostRequired},
		{"missing username", Config{Host: "h"}, ErrUsernameRequired},
		{"missing auth method", Config{Host: "h", Username: "u"}, ErrNoAuthMethod},
		{
			"strict checking without known hosts file",
			Config{Host: "h", Username: "u", Password: "p", StrictHostKeyChecking: true},
			ErrKnownHostsFileRequired,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := newClient(c.cfg, poisonDial(t))
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("newClient() error = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestNewClientWrapsPrivateKeyParseFailure(t *testing.T) {
	cfg := Config{Host: "h", Username: "u", PrivateKey: []byte("not a real key")}

	_, err := newClient(cfg, poisonDial(t))
	if err == nil {
		t.Fatal("newClient() = nil, want an error for an unparseable private key")
	}
}

func TestNewClientWrapsUnknownHostsFile(t *testing.T) {
	cfg := Config{
		Host: "h", Username: "u", Password: "p",
		StrictHostKeyChecking: true,
		KnownHostsFile:        "/nonexistent/known_hosts",
	}

	_, err := newClient(cfg, poisonDial(t))
	if err == nil {
		t.Fatal("newClient() = nil, want an error for a known_hosts file that does not exist")
	}
}

func TestNewClientWrapsDialFailure(t *testing.T) {
	dialErr := errors.New("connection refused")
	dial, called := fakeDial(nil, dialErr)

	_, err := newClient(validConfig(), dial)
	if !errors.Is(err, dialErr) {
		t.Fatalf("newClient() error = %v, want it to wrap %v", err, dialErr)
	}
	if !*called {
		t.Error("dial was never called for a valid Config")
	}
}

func TestNewClientSucceedsAndDefaultsTimeout(t *testing.T) {
	conn := &fakeConnection{session: &fakeSession{output: []byte("ok")}}
	dial, _ := fakeDial(conn, nil)

	c, err := newClient(validConfig(), dial)
	if err != nil {
		t.Fatalf("newClient() = %v", err)
	}
	if c.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want DefaultTimeout (%v) for a Config with no Timeout set", c.timeout, DefaultTimeout)
	}
}

func TestNewClientHonorsConfiguredTimeout(t *testing.T) {
	conn := &fakeConnection{session: &fakeSession{}}
	dial, _ := fakeDial(conn, nil)

	cfg := validConfig()
	cfg.Timeout = 5 * time.Second
	c, err := newClient(cfg, dial)
	if err != nil {
		t.Fatalf("newClient() = %v", err)
	}
	if c.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", c.timeout)
	}
}

// TestNewClientBuildsAuthMethodsFromPasswordAndPrivateKey proves both
// Password and PrivateKey, when both are set, are offered as candidate
// auth methods — see this package's doc comment, "Authentication."
func TestNewClientBuildsAuthMethodsFromPasswordAndPrivateKey(t *testing.T) {
	conn := &fakeConnection{session: &fakeSession{}}
	var gotAuthLen int
	dial := func(_ string, _ string, config *gossh.ClientConfig) (connection, error) {
		gotAuthLen = len(config.Auth)
		return conn, nil
	}

	cfg := validConfig()
	cfg.PrivateKey = generateTestPrivateKeyPEM(t)
	if _, err := newClient(cfg, dial); err != nil {
		t.Fatalf("newClient() = %v", err)
	}
	if gotAuthLen != 2 {
		t.Errorf("len(ClientConfig.Auth) = %d, want 2 (Password and PrivateKey both set)", gotAuthLen)
	}
}

func TestRunReturnsOutputOnSuccess(t *testing.T) {
	conn := &fakeConnection{session: &fakeSession{output: []byte("hello from device")}}
	c := &client{conn: conn, timeout: time.Second}

	got, err := c.Run(context.Background(), "show version")
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if got != "hello from device" {
		t.Errorf("Run() = %q, want %q", got, "hello from device")
	}
}

// TestRunTreatsNonZeroExitAsSuccess proves Run's documented contract:
// interpreting a remote command's exit status is out of scope, so an
// *ssh.ExitError does not surface as a Go error — the captured output is
// still returned.
func TestRunTreatsNonZeroExitAsSuccess(t *testing.T) {
	conn := &fakeConnection{session: &fakeSession{
		output: []byte("partial output before nonzero exit"),
		err:    &gossh.ExitError{},
	}}
	c := &client{conn: conn, timeout: time.Second}

	got, err := c.Run(context.Background(), "exit 1")
	if err != nil {
		t.Fatalf("Run() = %v, want nil error for a nonzero exit status", err)
	}
	if got != "partial output before nonzero exit" {
		t.Errorf("Run() = %q, want the captured output", got)
	}
}

// TestRunTreatsExitMissingAsSuccess is
// TestRunTreatsNonZeroExitAsSuccess's counterpart for a session torn
// down without ever reporting an exit status at all.
func TestRunTreatsExitMissingAsSuccess(t *testing.T) {
	conn := &fakeConnection{session: &fakeSession{
		output: []byte("output"),
		err:    &gossh.ExitMissingError{},
	}}
	c := &client{conn: conn, timeout: time.Second}

	got, err := c.Run(context.Background(), "cmd")
	if err != nil {
		t.Fatalf("Run() = %v, want nil error when no exit status was reported", err)
	}
	if got != "output" {
		t.Errorf("Run() = %q, want %q", got, "output")
	}
}

func TestRunWrapsGenuineSessionFailure(t *testing.T) {
	sessionErr := errors.New("broken pipe")
	conn := &fakeConnection{session: &fakeSession{err: sessionErr}}
	c := &client{conn: conn, timeout: time.Second}

	_, err := c.Run(context.Background(), "cmd")
	if !errors.Is(err, sessionErr) {
		t.Fatalf("Run() error = %v, want it to wrap %v", err, sessionErr)
	}
}

func TestRunWrapsNewSessionFailure(t *testing.T) {
	sessionErr := errors.New("no more sessions")
	conn := &fakeConnection{newSessionErr: sessionErr}
	c := &client{conn: conn, timeout: time.Second}

	_, err := c.Run(context.Background(), "cmd")
	if !errors.Is(err, sessionErr) {
		t.Fatalf("Run() error = %v, want it to wrap %v", err, sessionErr)
	}
}

// TestRunHonorsContextCancellation proves Run aborts (and closes the
// session, to unblock the underlying goroutine) as soon as ctx is
// cancelled, rather than waiting for a blocked command to finish.
func TestRunHonorsContextCancellation(t *testing.T) {
	fs := &fakeSession{block: make(chan struct{}), outputted: make(chan struct{})}
	conn := &fakeConnection{session: fs}
	c := &client{conn: conn, timeout: time.Minute} // long enough that only cancellation could end this

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := c.Run(ctx, "cmd")
		done <- err
	}()

	<-fs.outputted // the fake command is now "running"
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() error = %v, want it to wrap context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after ctx was cancelled")
	}
	if !fs.closed {
		t.Error("Run() did not close the session after cancellation")
	}
	close(fs.block) // release the blocked goroutine so the test process can exit cleanly
}

// TestRunHonorsConfiguredTimeout is
// TestRunHonorsContextCancellation's counterpart for Config.Timeout
// rather than caller-driven cancellation.
func TestRunHonorsConfiguredTimeout(t *testing.T) {
	fs := &fakeSession{block: make(chan struct{})}
	conn := &fakeConnection{session: fs}
	c := &client{conn: conn, timeout: 10 * time.Millisecond}

	_, err := c.Run(context.Background(), "cmd")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
	close(fs.block)
}

func TestRunReturnsErrClientClosedAfterClose(t *testing.T) {
	conn := &fakeConnection{session: &fakeSession{}}
	c := &client{conn: conn, timeout: time.Second}

	if err := c.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}

	_, err := c.Run(context.Background(), "cmd")
	if !errors.Is(err, ErrClientClosed) {
		t.Fatalf("Run() error = %v, want %v", err, ErrClientClosed)
	}
}

func TestCloseDelegatesToConnection(t *testing.T) {
	closeErr := errors.New("close failed")
	conn := &fakeConnection{closeErr: closeErr}
	c := &client{conn: conn, timeout: time.Second}

	err := c.Close()
	if !errors.Is(err, closeErr) {
		t.Fatalf("Close() = %v, want %v", err, closeErr)
	}
	if !conn.closed {
		t.Error("Close() did not close the underlying connection")
	}
}
