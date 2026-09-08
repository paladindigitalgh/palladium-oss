package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// interactiveSession is the subset of *golang.org/x/crypto/ssh.Session
// this package depends on for interactive shell mode — a PTY request, a
// stdin/stdout pipe pair, and a shell request, as opposed to session's
// (client.go) narrower Output-only view for one-shot exec commands.
// *gossh.Session satisfies it structurally, the same as session; see
// connection.NewInteractiveSession in client.go for the one place that
// distinction requires an adapter.
type interactiveSession interface {
	RequestPty(term string, height, width int, modes gossh.TerminalModes) error
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.Reader, error)
	Shell() error
	Close() error
}

const (
	// defaultTerm, defaultTermHeight, and defaultTermWidth describe the
	// PTY this package requests. Nothing in this milestone's scope needs
	// these to be configurable — a wide, generous terminal minimizes the
	// chance of a device's own line-wrapping corrupting captured output,
	// and no target device has yet needed a specific terminal type.
	defaultTerm       = "vt100"
	defaultTermHeight = 50
	defaultTermWidth  = 200

	// promptIdleWindow is how long the initial prompt-detection read
	// (see detectInitialPrompt) waits for the remote side to go quiet
	// before treating whatever is at the end of the buffer as a
	// candidate prompt line. This is only used once, immediately after a
	// Shell is opened, before any command has been sent — every
	// RunCommand afterward matches against promptPattern, derived once
	// from that detected line, which needs no idle timing at all (see
	// matchPrompt).
	promptIdleWindow = 300 * time.Millisecond

	// promptDetectionTimeout bounds the entire initial prompt-detection
	// wait — see ErrPromptNotDetected.
	promptDetectionTimeout = 10 * time.Second
)

// promptLinePattern is a best-effort heuristic for recognizing a shell
// prompt's own line during initial detection: a short, non-blank line
// ending in '#' or '>', the shape shared by the overwhelming majority of
// Cisco-style, DASAN-style, and Junos-style network CLIs (confirmed
// firsthand against a real Kontron/Iskratel C16 OLT prompt,
// "PineCreek-OLT01#", during this package's development). This is a
// generic heuristic about prompt shape, not knowledge of any specific
// vendor — see this package's doc comment, "Interactive shell mode."
var promptLinePattern = regexp.MustCompile(`\S.*[#>]$`)

// shell is Shell's one implementation.
type shell struct {
	sess    interactiveSession
	stdin   io.WriteCloser
	timeout time.Duration
	// prompt is the exact, literal prompt line detected once at
	// connection time (see detectInitialPrompt) — kept for diagnostics
	// and tests. RunCommand itself never matches against this literal
	// string directly; see promptPattern for why.
	prompt string
	// promptPattern is derived from prompt's hostname portion (see
	// extractHostname) and is what RunCommand actually matches against.
	// A literal-string match would break the instant any command changes
	// the prompt's mode suffix — confirmed firsthand against a real
	// Kontron/Iskratel C16, which changes its prompt at every config-mode
	// transition: "test-olt#" -> "test-olt(Config)#" on "configure", then
	// -> "test-olt(Interface xgs/1/2)#" on "interface xgs/1/2" — and
	// RunCommand hung until timeout on each, waiting for a prompt shape
	// that could never reappear. The mode suffix is not just non-
	// whitespace text, either: "(Interface xgs/1/2)" itself contains a
	// space, which is why this matches "the hostname, then anything up
	// to the next # or > on that line" rather than restricting the
	// suffix to \S* — a first attempt at this fix used \S* and still hung
	// on exactly this prompt, since it could not span the space before
	// "xgs/1/2". promptPattern instead recognizes any of these
	// transitions as "the prompt is back," not just the exact shape
	// first seen.
	promptPattern *regexp.Regexp
	closed        bool

	// chunks and readErr are fed by the single background goroutine
	// (see readLoop) that owns the only Read call ever made against
	// this Shell's stdout pipe. Both detectInitialPrompt (once, during
	// newShell) and every later RunCommand read from these same two
	// channels — never from stdout directly — because a Shell, like a
	// Client, is not safe for concurrent use, but its one background
	// reader must still run continuously across RunCommand's separate
	// calls rather than being restarted for each one.
	chunks  chan []byte
	readErr chan error

	// done and stopOnce let Close (or a failed newShellWithTiming) tell
	// readLoop to abandon a chunk it is trying to hand off, rather than
	// block forever on an unbuffered send once nothing is left to
	// receive it — e.g. after detectInitialPrompt gives up and returns,
	// or after a RunCommand caller's ctx is cancelled and no further
	// RunCommand call ever arrives to drain sh.chunks again. Closing the
	// underlying session alone is not enough for this: it unblocks a
	// pending Read, but readLoop may already be past that point and
	// blocked on the channel send instead.
	done     chan struct{}
	stopOnce sync.Once
}

var _ Shell = (*shell)(nil)

// newShell opens sess as a PTY-backed interactive shell and blocks until
// that device's own command prompt is detected, using
// promptIdleWindow and promptDetectionTimeout. See newShellWithTiming,
// which this delegates to, for why those two are a separate parameter on
// an unexported entry point rather than baked into this function
// directly — the same reasoning newClient documents for taking a
// dialFunc parameter New itself does not expose.
func newShell(ctx context.Context, sess interactiveSession, timeout time.Duration) (*shell, error) {
	return newShellWithTiming(ctx, sess, timeout, promptIdleWindow, promptDetectionTimeout)
}

func newShellWithTiming(ctx context.Context, sess interactiveSession, timeout, idleWindow, detectionTimeout time.Duration) (*shell, error) {
	// ECHO is disabled on the PTY we request, but many embedded network
	// CLIs implement their own command-line editing above the tty layer
	// and echo typed input regardless of this setting — see Shell's own
	// doc comment on RunCommand for why this package does not try to
	// strip that echo back out.
	modes := gossh.TerminalModes{
		gossh.ECHO:          0,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}
	if err := sess.RequestPty(defaultTerm, defaultTermHeight, defaultTermWidth, modes); err != nil {
		_ = sess.Close()
		return nil, fmt.Errorf("ssh: request pty: %w", err)
	}

	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		return nil, fmt.Errorf("ssh: open interactive shell stdin: %w", err)
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		return nil, fmt.Errorf("ssh: open interactive shell stdout: %w", err)
	}
	if err := sess.Shell(); err != nil {
		_ = sess.Close()
		return nil, fmt.Errorf("ssh: start interactive shell: %w", err)
	}

	sh := &shell{
		sess:    sess,
		stdin:   stdin,
		timeout: timeout,
		chunks:  make(chan []byte),
		readErr: make(chan error, 1),
		done:    make(chan struct{}),
	}
	go sh.readLoop(stdout)

	prompt, err := sh.detectInitialPrompt(ctx, idleWindow, detectionTimeout)
	if err != nil {
		sh.stop()
		_ = sess.Close()
		return nil, err
	}
	sh.prompt = prompt
	sh.promptPattern = regexp.MustCompile(regexp.QuoteMeta(extractHostname(prompt)) + `[^\r\n]*[#>]\s*$`)

	return sh, nil
}

// stop signals readLoop to abandon a pending chunk send rather than
// block forever — see the doc comment on shell.done — and is safe to
// call more than once (Close does not otherwise guard against being
// called twice, mirroring Client.Close's own documented behavior).
func (sh *shell) stop() {
	sh.stopOnce.Do(func() { close(sh.done) })
}

// readLoop is the one and only goroutine that ever calls stdout.Read. It
// runs until stdout.Read fails (typically io.EOF, once sess.Close is
// called) or sh.done is closed, forwarding every chunk read to
// sh.chunks and reporting the terminal read error, if any, on
// sh.readErr before closing sh.chunks.
func (sh *shell) readLoop(stdout io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case sh.chunks <- chunk:
			case <-sh.done:
				return
			}
		}
		if err != nil {
			sh.readErr <- err
			close(sh.chunks)
			return
		}
	}
}

// detectInitialPrompt accumulates output from sh.chunks until the
// remote side has gone quiet for idleWindow, then checks whether the
// last non-blank line looks like a prompt (see promptLinePattern). It
// keeps waiting through further idle windows — a login banner may print
// in more than one burst — until detectionTimeout elapses, at which
// point it gives up with ErrPromptNotDetected.
func (sh *shell) detectInitialPrompt(ctx context.Context, idleWindow, detectionTimeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, detectionTimeout)
	defer cancel()

	var buf bytes.Buffer
	idle := time.NewTimer(idleWindow)
	defer idle.Stop()

	for {
		select {
		case chunk, ok := <-sh.chunks:
			if !ok {
				return "", fmt.Errorf("ssh: interactive shell closed while waiting for initial prompt: %w", <-sh.readErr)
			}
			buf.Write(chunk)
			if !idle.Stop() {
				<-idle.C
			}
			idle.Reset(idleWindow)

		case err := <-sh.readErr:
			return "", fmt.Errorf("ssh: interactive shell read failed while waiting for initial prompt: %w", err)

		case <-idle.C:
			if prompt, ok := lastPromptLine(buf.String()); ok {
				return prompt, nil
			}
			idle.Reset(idleWindow)

		case <-ctx.Done():
			return "", ErrPromptNotDetected
		}
	}
}

// lastPromptLine returns the last non-blank line of buf, and whether it
// looks like a prompt (see promptLinePattern). A non-blank last line
// that does not look like a prompt means the remote side is still
// settling (e.g. mid-banner) rather than genuinely idle at a prompt.
func lastPromptLine(buf string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(buf, "\r\n", "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimRight(lines[i], " \t")
		if line == "" {
			continue
		}
		return line, promptLinePattern.MatchString(line)
	}
	return "", false
}

// extractHostname returns promptLine's hostname portion — the part
// before any parenthetical mode suffix (e.g. "(Config)", "(Config-if)")
// and before its trailing '#' or '>' — after trimming any leading
// whitespace or stray '\r' a device's own prompt redraw may prepend
// (confirmed firsthand: a real Kontron/Iskratel C16 sent "\rHOSTNAME>"
// as its very first detected prompt line). "HOSTNAME>", "HOSTNAME#", and
// "HOSTNAME(Config)#" all yield "HOSTNAME" — see promptPattern's own
// doc comment on why matching stays anchored to this portion alone
// across every mode transition.
func extractHostname(promptLine string) string {
	trimmed := strings.TrimLeft(promptLine, "\r\n\t ")
	if idx := strings.IndexByte(trimmed, '('); idx != -1 {
		return trimmed[:idx]
	}
	return strings.TrimRight(trimmed, "#> \t")
}

// matchPrompt reports whether buf ends (allowing only trailing
// whitespace after) with a reappearance of sh's prompt — matched via
// promptPattern (hostname plus any mode suffix, ending in '#' or '>'),
// not an exact literal string, so a command that changes the prompt's
// mode suffix (see promptPattern's own doc comment) is still recognized
// as "the prompt is back." On a match it returns everything before that
// trailing prompt occurrence.
func (sh *shell) matchPrompt(buf string) (string, bool) {
	loc := sh.promptPattern.FindStringIndex(buf)
	if loc == nil {
		return "", false
	}
	return buf[:loc[0]], true
}

// stripPagerTrigger looks for the first (in argument order) PagerPrompt
// among pagers whose Trigger appears anywhere in buf. On a match it
// returns buf with that one occurrence of Trigger removed, that
// PagerPrompt's Response, and true.
func stripPagerTrigger(buf string, pagers []PagerPrompt) (stripped, response string, ok bool) {
	for _, p := range pagers {
		if idx := strings.Index(buf, p.Trigger); idx != -1 {
			return buf[:idx] + buf[idx+len(p.Trigger):], p.Response, true
		}
	}
	return buf, "", false
}

// RunCommand implements Shell.
func (sh *shell) RunCommand(ctx context.Context, command string, pagers ...PagerPrompt) (string, error) {
	if sh.closed {
		return "", ErrShellClosed
	}

	if _, err := io.WriteString(sh.stdin, command+"\n"); err != nil {
		return "", fmt.Errorf("ssh: write command to interactive shell: %w", err)
	}

	runCtx := ctx
	if sh.timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, sh.timeout)
		defer cancel()
	}

	var buf bytes.Buffer
	for {
		select {
		case chunk, ok := <-sh.chunks:
			if !ok {
				return "", fmt.Errorf("ssh: interactive shell closed: %w", <-sh.readErr)
			}
			buf.Write(chunk)

			// Checked before the prompt: a pager prompt is itself a
			// short line that could otherwise be mistaken for the real
			// one if it happened to end the same way, so any pager
			// hit is handled — and removed from the buffer entirely —
			// before matchPrompt ever looks at this chunk.
			if stripped, response, hit := stripPagerTrigger(buf.String(), pagers); hit {
				buf.Reset()
				buf.WriteString(stripped)
				if _, err := io.WriteString(sh.stdin, response); err != nil {
					return "", fmt.Errorf("ssh: write pager response to interactive shell: %w", err)
				}
				continue
			}

			if out, done := sh.matchPrompt(buf.String()); done {
				return out, nil
			}

		case err := <-sh.readErr:
			return "", fmt.Errorf("ssh: interactive shell read failed: %w", err)

		case <-runCtx.Done():
			return "", fmt.Errorf("ssh: interactive shell run command: %w", runCtx.Err())
		}
	}
}

// Close implements Shell.
func (sh *shell) Close() error {
	sh.closed = true
	sh.stop()
	return sh.sess.Close()
}
