package ssh

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// fakeInteractiveSession is an in-memory interactiveSession, backed by a
// pair of in-process pipes so a test can act as "the device": read
// whatever the Shell under test writes to stdin, and write back whatever
// output a real device would have produced, on its own schedule — the
// same reason connection/session (client.go) exist as interfaces at all,
// applied to interactive shell mode.
type fakeInteractiveSession struct {
	ptyErr    error
	stdinErr  error
	stdoutErr error
	shellErr  error
	closeErr  error
	closed    bool

	stdinR  *io.PipeReader
	stdinW  *io.PipeWriter
	stdoutR *io.PipeReader
	stdoutW *io.PipeWriter
}

func newFakeInteractiveSession() *fakeInteractiveSession {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	return &fakeInteractiveSession{stdinR: stdinR, stdinW: stdinW, stdoutR: stdoutR, stdoutW: stdoutW}
}

func (s *fakeInteractiveSession) RequestPty(string, int, int, gossh.TerminalModes) error {
	return s.ptyErr
}

func (s *fakeInteractiveSession) StdinPipe() (io.WriteCloser, error) {
	if s.stdinErr != nil {
		return nil, s.stdinErr
	}
	return s.stdinW, nil
}

func (s *fakeInteractiveSession) StdoutPipe() (io.Reader, error) {
	if s.stdoutErr != nil {
		return nil, s.stdoutErr
	}
	return s.stdoutR, nil
}

func (s *fakeInteractiveSession) Shell() error { return s.shellErr }

func (s *fakeInteractiveSession) Close() error {
	s.closed = true
	_ = s.stdoutW.Close()
	_ = s.stdinR.Close()
	return s.closeErr
}

// readLine reads one newline-terminated command off stdinR, as a
// simulated device would after a human (or a Shell) presses enter — used
// by tests playing the "device" side of the fake.
func (s *fakeInteractiveSession) readLine(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 0, 64)
	one := make([]byte, 1)
	for {
		n, err := s.stdinR.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				return string(buf)
			}
			buf = append(buf, one[0])
		}
		if err != nil {
			t.Fatalf("test device: read command: %v", err)
		}
	}
}

const testTiming = 20 * time.Millisecond

func TestInteractiveDetectsPromptAndRunsCommand(t *testing.T) {
	sess := newFakeInteractiveSession()

	device := make(chan struct{})
	go func() {
		defer close(device)
		// Simulated login banner followed by the initial prompt, sent
		// in two separate writes to prove detectInitialPrompt tolerates
		// a banner arriving in more than one burst before settling.
		_, _ = io.WriteString(sess.stdoutW, "Welcome to PineCreek-OLT01\r\n")
		time.Sleep(testTiming / 2)
		_, _ = io.WriteString(sess.stdoutW, "PineCreek-OLT01#")

		cmd := sess.readLine(t)
		if cmd != "show sysinfo" {
			t.Errorf("test device received command %q, want %q", cmd, "show sysinfo")
		}
		_, _ = io.WriteString(sess.stdoutW, "show sysinfo\r\nUptime: 12 days\r\nPineCreek-OLT01#")
	}()

	sh, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}
	t.Cleanup(func() { _ = sh.Close() })
	if sh.prompt != "PineCreek-OLT01#" {
		t.Errorf("detected prompt = %q, want %q", sh.prompt, "PineCreek-OLT01#")
	}

	out, err := sh.RunCommand(context.Background(), "show sysinfo")
	if err != nil {
		t.Fatalf("RunCommand() = %v", err)
	}
	if !strings.Contains(out, "Uptime: 12 days") {
		t.Errorf("RunCommand() = %q, want it to contain %q", out, "Uptime: 12 days")
	}
	if strings.Contains(out, "PineCreek-OLT01#") {
		t.Errorf("RunCommand() = %q, want the trailing prompt excluded", out)
	}

	<-device
}

// TestInteractiveRunCommandFollowsPromptModeChange proves the actual bug
// fix: a command that changes the device's prompt (elevating from
// "HOSTNAME>" to "HOSTNAME#" via "enable", the same transition confirmed
// firsthand to hang indefinitely against a real Kontron/Iskratel C16
// before this fix) is still recognized as "the prompt is back," not
// mistaken for a hang. A literal-string match (the pre-fix behavior)
// would wait forever here, since "PineCreek-OLT01>" never reappears once
// the device elevates to "PineCreek-OLT01#".
func TestInteractiveRunCommandFollowsPromptModeChange(t *testing.T) {
	sess := newFakeInteractiveSession()

	device := make(chan struct{})
	go func() {
		defer close(device)
		_, _ = io.WriteString(sess.stdoutW, "PineCreek-OLT01>")

		cmd := sess.readLine(t)
		if cmd != "enable" {
			t.Errorf("command = %q, want %q", cmd, "enable")
		}
		// Silent elevation: no password, no output, just a new prompt
		// ending in "#" instead of ">".
		_, _ = io.WriteString(sess.stdoutW, "enable\r\nPineCreek-OLT01#")
	}()

	sh, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}
	t.Cleanup(func() { _ = sh.Close() })

	out, err := sh.RunCommand(context.Background(), "enable")
	if err != nil {
		t.Fatalf("RunCommand() = %v, want it to return once the elevated prompt appears, not hang", err)
	}
	if strings.Contains(out, "PineCreek-OLT01#") {
		t.Errorf("RunCommand() = %q, want the trailing prompt excluded", out)
	}

	<-device
}

// TestInteractiveRunCommandFollowsConfigModePrompt proves the same fix
// for entering a parenthetical config-mode prompt suffix
// ("HOSTNAME#" -> "HOSTNAME(Config)#"), the transition "configure"
// itself triggers.
func TestInteractiveRunCommandFollowsConfigModePrompt(t *testing.T) {
	sess := newFakeInteractiveSession()

	device := make(chan struct{})
	go func() {
		defer close(device)
		_, _ = io.WriteString(sess.stdoutW, "PineCreek-OLT01#")

		cmd := sess.readLine(t)
		if cmd != "configure" {
			t.Errorf("command = %q, want %q", cmd, "configure")
		}
		_, _ = io.WriteString(sess.stdoutW, "configure\r\nPineCreek-OLT01(Config)#")
	}()

	sh, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}
	t.Cleanup(func() { _ = sh.Close() })

	out, err := sh.RunCommand(context.Background(), "configure")
	if err != nil {
		t.Fatalf("RunCommand() = %v, want it to return once the config-mode prompt appears, not hang", err)
	}
	if strings.Contains(out, "PineCreek-OLT01(Config)#") {
		t.Errorf("RunCommand() = %q, want the trailing prompt excluded", out)
	}

	<-device
}

// TestInteractiveRunCommandFollowsPromptWithSpaceInModeSuffix proves the
// fix for the actual real-hardware failure: a real Kontron/Iskratel C16
// changes its prompt to "test-olt(Interface xgs/1/2)#" on
// "interface xgs/1/2" — a mode suffix containing a space, not just
// non-whitespace text. A first attempt at promptPattern used `\S*` for
// the suffix and hung on exactly this prompt, unable to span the space
// before "xgs/1/2".
func TestInteractiveRunCommandFollowsPromptWithSpaceInModeSuffix(t *testing.T) {
	sess := newFakeInteractiveSession()

	device := make(chan struct{})
	go func() {
		defer close(device)
		_, _ = io.WriteString(sess.stdoutW, "test-olt(Config)#")

		cmd := sess.readLine(t)
		if cmd != "interface xgs/1/2" {
			t.Errorf("command = %q, want %q", cmd, "interface xgs/1/2")
		}
		_, _ = io.WriteString(sess.stdoutW, "interface xgs/1/2\r\ntest-olt(Interface xgs/1/2)#")
	}()

	sh, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}
	t.Cleanup(func() { _ = sh.Close() })

	out, err := sh.RunCommand(context.Background(), "interface xgs/1/2")
	if err != nil {
		t.Fatalf("RunCommand() = %v, want it to return once the space-containing prompt appears, not hang", err)
	}
	if strings.Contains(out, "test-olt(Interface xgs/1/2)#") {
		t.Errorf("RunCommand() = %q, want the trailing prompt excluded", out)
	}

	<-device
}

// TestInteractiveRunCommandReusesSameShellAcrossCalls proves a Shell is
// one long-lived process, not a fresh one per RunCommand: two commands
// in a row, each waiting for the same literal prompt.
func TestInteractiveRunCommandReusesSameShellAcrossCalls(t *testing.T) {
	sess := newFakeInteractiveSession()

	device := make(chan struct{})
	go func() {
		defer close(device)
		_, _ = io.WriteString(sess.stdoutW, "PineCreek-OLT01#")

		for i, want := range []string{"show version", "show onu interface xgs/1/1"} {
			cmd := sess.readLine(t)
			if cmd != want {
				t.Errorf("command #%d = %q, want %q", i+1, cmd, want)
			}
			_, _ = io.WriteString(sess.stdoutW, cmd+"\r\nresponse "+cmd+"\r\nPineCreek-OLT01#")
		}
	}()

	sh, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}
	t.Cleanup(func() { _ = sh.Close() })

	out1, err := sh.RunCommand(context.Background(), "show version")
	if err != nil {
		t.Fatalf("RunCommand(#1) = %v", err)
	}
	if !strings.Contains(out1, "response show version") {
		t.Errorf("RunCommand(#1) = %q, want it to contain the first response", out1)
	}

	out2, err := sh.RunCommand(context.Background(), "show onu interface xgs/1/1")
	if err != nil {
		t.Fatalf("RunCommand(#2) = %v", err)
	}
	if !strings.Contains(out2, "response show onu interface xgs/1/1") {
		t.Errorf("RunCommand(#2) = %q, want it to contain the second response", out2)
	}

	<-device
}

// TestInteractiveRunCommandHandlesMidCommandPager proves RunCommand
// transparently steps past a device's own pager prompt(s) — confirmed
// firsthand on a real Kontron/Iskratel C16 for long single-ONU detail
// output — sending each PagerPrompt's Response and excluding both the
// trigger text and the response from the returned output.
func TestInteractiveRunCommandHandlesMidCommandPager(t *testing.T) {
	sess := newFakeInteractiveSession()
	pager := PagerPrompt{
		Trigger:  "Press any key to continue, ESC to stop scrolling or TAB to scroll to the end.",
		Response: " ",
	}

	device := make(chan struct{})
	go func() {
		defer close(device)
		_, _ = io.WriteString(sess.stdoutW, "PineCreek-OLT01#")

		if cmd := sess.readLine(t); cmd != "show onu interface xgs/1/1" {
			t.Errorf("command = %q, want %q", cmd, "show onu interface xgs/1/1")
		}
		_, _ = io.WriteString(sess.stdoutW, "Interface   xgs/1/1\r\n"+pager.Trigger)

		// The pager keystroke arrives on stdin, not as a newline-
		// terminated command — read exactly one byte to confirm it's
		// the pager's own Response, then finish the command's output.
		got := make([]byte, 1)
		if _, err := io.ReadFull(sess.stdinR, got); err != nil {
			t.Errorf("test device: read pager response: %v", err)
			return
		}
		if string(got) != pager.Response {
			t.Errorf("pager response = %q, want %q", got, pager.Response)
		}
		_, _ = io.WriteString(sess.stdoutW, "Rx Power [dBm]   -19.67\r\nPineCreek-OLT01#")
	}()

	sh, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}
	t.Cleanup(func() { _ = sh.Close() })

	out, err := sh.RunCommand(context.Background(), "show onu interface xgs/1/1", pager)
	if err != nil {
		t.Fatalf("RunCommand() = %v", err)
	}
	if !strings.Contains(out, "Interface   xgs/1/1") || !strings.Contains(out, "Rx Power [dBm]   -19.67") {
		t.Errorf("RunCommand() = %q, want it to contain both halves of the paged output", out)
	}
	if strings.Contains(out, pager.Trigger) {
		t.Errorf("RunCommand() = %q, want the pager trigger text excluded", out)
	}
	if strings.Contains(out, "PineCreek-OLT01#") {
		t.Errorf("RunCommand() = %q, want the trailing prompt excluded", out)
	}

	<-device
}

func TestInteractiveReturnsErrPromptNotDetectedWhenDeviceNeverSettles(t *testing.T) {
	sess := newFakeInteractiveSession()

	go func() {
		// Never anything prompt-shaped, ever — just noise.
		_, _ = io.WriteString(sess.stdoutW, "still booting...\r\n")
	}()

	_, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 3*testTiming)
	if !errors.Is(err, ErrPromptNotDetected) {
		t.Fatalf("newShellWithTiming() error = %v, want %v", err, ErrPromptNotDetected)
	}
	if !sess.closed {
		t.Error("newShellWithTiming() did not close the session after failing to detect a prompt")
	}
}

func TestInteractiveRunCommandReturnsErrShellClosedAfterClose(t *testing.T) {
	sess := newFakeInteractiveSession()
	go func() { _, _ = io.WriteString(sess.stdoutW, "PineCreek-OLT01#") }()

	sh, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}

	if err := sh.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if !sess.closed {
		t.Error("Close() did not close the underlying session")
	}

	if _, err := sh.RunCommand(context.Background(), "show version"); !errors.Is(err, ErrShellClosed) {
		t.Fatalf("RunCommand() error = %v, want %v", err, ErrShellClosed)
	}
}

// TestInteractiveRunCommandHonorsContextCancellation proves RunCommand
// aborts as soon as ctx is cancelled rather than waiting forever for a
// prompt that never reappears — the interactive-shell counterpart of
// TestRunHonorsContextCancellation in client_test.go.
func TestInteractiveRunCommandHonorsContextCancellation(t *testing.T) {
	sess := newFakeInteractiveSession()
	go func() {
		_, _ = io.WriteString(sess.stdoutW, "PineCreek-OLT01#")
		_ = sess.readLine(t) // consume the command; deliberately never reply
	}()

	sh, err := newShellWithTiming(context.Background(), sess, time.Minute, testTiming, 2*time.Second)
	if err != nil {
		t.Fatalf("newShellWithTiming() = %v", err)
	}
	t.Cleanup(func() { _ = sh.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := sh.RunCommand(ctx, "show version")
		done <- err
	}()

	time.Sleep(testTiming) // give RunCommand time to write the command and start waiting
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("RunCommand() error = %v, want it to wrap context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunCommand() did not return after ctx was cancelled")
	}
}

func TestInteractiveWrapsRequestPtyFailure(t *testing.T) {
	sess := newFakeInteractiveSession()
	sess.ptyErr = errors.New("pty rejected")

	_, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, time.Second)
	if !errors.Is(err, sess.ptyErr) {
		t.Fatalf("newShellWithTiming() error = %v, want it to wrap %v", err, sess.ptyErr)
	}
	if !sess.closed {
		t.Error("newShellWithTiming() did not close the session after a RequestPty failure")
	}
}

func TestInteractiveWrapsShellFailure(t *testing.T) {
	sess := newFakeInteractiveSession()
	sess.shellErr = errors.New("shell rejected")

	_, err := newShellWithTiming(context.Background(), sess, time.Second, testTiming, time.Second)
	if !errors.Is(err, sess.shellErr) {
		t.Fatalf("newShellWithTiming() error = %v, want it to wrap %v", err, sess.shellErr)
	}
}

func TestClientInteractiveReturnsErrClientClosedAfterClose(t *testing.T) {
	conn := &fakeConnection{}
	c := &client{conn: conn, timeout: time.Second}

	if err := c.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}

	if _, err := c.Interactive(context.Background()); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("Interactive() error = %v, want %v", err, ErrClientClosed)
	}
}

func TestClientInteractiveWrapsNewInteractiveSessionFailure(t *testing.T) {
	sessionErr := errors.New("no more sessions")
	conn := &fakeConnection{newInteractiveSessionErr: sessionErr}
	c := &client{conn: conn, timeout: time.Second}

	_, err := c.Interactive(context.Background())
	if !errors.Is(err, sessionErr) {
		t.Fatalf("Interactive() error = %v, want it to wrap %v", err, sessionErr)
	}
}

// TestExtractHostname covers the exact prompt shapes confirmed firsthand
// against a real Kontron/Iskratel C16: a leading '\r' its own prompt
// redraw prepends, plain user/privileged EXEC prompts, and a
// parenthetical config-mode suffix.
func TestExtractHostname(t *testing.T) {
	cases := []struct {
		promptLine string
		want       string
	}{
		{"\rtest-olt>", "test-olt"},
		{"test-olt#", "test-olt"},
		{"test-olt(Config)#", "test-olt"},
		{"test-olt(Config-if)#", "test-olt"},
		{"PineCreek-OLT01#", "PineCreek-OLT01"},
	}
	for _, tc := range cases {
		if got := extractHostname(tc.promptLine); got != tc.want {
			t.Errorf("extractHostname(%q) = %q, want %q", tc.promptLine, got, tc.want)
		}
	}
}
