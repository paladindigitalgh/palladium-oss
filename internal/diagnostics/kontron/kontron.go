// Package kontron implements Palladium's first real, vendor-specific
// command surface: the actual `show` commands a Kontron/Iskratel C16
// OLT understands, run over an already-open interactive
// internal/platform/ssh.Shell — that package's "Interactive shell mode"
// exists specifically because a real C16 was confirmed, firsthand, not
// to support the plain SSH exec channel internal/platform/ssh.Client.Run
// uses.
//
// Per CLAUDE.md's Plugin Philosophy ("The core system must never
// contain Kontron-...specific logic"), this package is deliberately
// narrow: it knows the exact command strings this OLT's CLI expects and
// how to recognize (and step past) its pager — nothing else. Every
// method returns the device's raw text, verbatim, no different from
// internal/platform/ssh.Client.Run's own "no command parsing" boundary.
// Commands are added one at a time, each backed by real sample output
// captured against a real production C16.
//
// # What is deliberately not here yet
//
//   - No connecting. A Client is handed an already-open ssh.Shell —
//     resolving a specific OLT (via internal/olt) to a reachable
//     address and credentials (via internal/connectionprofile and
//     internal/authentication) and calling ssh.New/Interactive to open
//     that Shell is a separate concern, not this package's job.
//   - No wiring into internal/diagnostics's Diagnostic/Registry/HTTP
//     framework. That framework's Request identifies work by a single
//     ONU UUID, which assumes an ONU domain and OLT topology resolution
//     that do not exist yet in this codebase (see
//     internal/diagnostics/diagnostic.go's own doc comment). Wiring this
//     package into that framework is a deliberate future step, not
//     implied by anything here.
//   - No output parsing, and therefore no ONU/interface data model of
//     its own — every method's return value is exactly what the device
//     printed for that command.
package kontron

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// ErrInvalidInterface means an iface argument (e.g. "xgs/1/1") passed to
// one of Client's per-interface methods contained a newline or carriage
// return.
//
// This is deliberately not full format validation of what a real
// interface identifier looks like — iface is expected to come from
// Palladium's own stored data (an OLT or ONU record) rather than a user
// typing it into a form, so there is no untrusted free text to validate
// here in the usual sense. But every iface value is interpolated
// directly into a command line written to the device's interactive
// shell (see internal/platform/ssh's "Interactive shell mode" doc
// comment), which treats each newline-terminated line as a separate
// command — so a stray newline in iface is a command-injection shape,
// not a formatting nitpick, and is rejected regardless of how trusted
// the immediate caller is assumed to be.
var ErrInvalidInterface = errors.New("kontron: interface value contains a newline")

// pager is this platform's own pager prompt, confirmed firsthand against
// real production C16 output: a single command's output longer than the
// device's configured printout-limit (25 lines, by default) is broken up
// mid-command by this exact text, dismissed by any keypress. A single
// space is sent back — the traditional "continue paging" keystroke.
//
// This is supplied to every command below regardless of whether that
// specific command's output is normally short enough to avoid triggering
// it: harmless when it never fires, and this platform's printout-limit
// is a per-OLT, operator-controlled setting (see
// internal/platform/ssh's own "Interactive shell mode" doc comment) this
// package cannot assume has been raised on every OLT it will ever run
// against.
var pager = ssh.PagerPrompt{
	Trigger:  "Press any key to continue, ESC to stop scrolling or TAB to scroll to the end.",
	Response: " ",
}

// Client runs Kontron/Iskratel C16 show commands over an already-open
// interactive shell.
type Client struct {
	shell ssh.Shell
}

// NewClient builds a Client around shell, which must already be open
// (see ssh.Client.Interactive) and logged in to the target OLT. NewClient
// does not take ownership of shell's lifecycle — the caller remains
// responsible for closing it once done with every Client method call
// that will be made against it.
func NewClient(shell ssh.Shell) *Client {
	return &Client{shell: shell}
}

// run executes command and wraps any failure with the command itself,
// so an error surfaced further up the stack says what was actually run.
func (c *Client) run(ctx context.Context, command string) (string, error) {
	out, err := c.shell.RunCommand(ctx, command, pager)
	if err != nil {
		return "", fmt.Errorf("kontron: %s: %w", command, err)
	}
	return out, nil
}

// runForInterface builds a command from template (a single "%s"
// verb standing in for the interface) and iface, validating iface
// first (see ErrInvalidInterface), then delegates to run.
func (c *Client) runForInterface(ctx context.Context, template, iface string) (string, error) {
	if strings.ContainsAny(iface, "\n\r") {
		return "", ErrInvalidInterface
	}
	return c.run(ctx, fmt.Sprintf(template, iface))
}

// onuSummaryCommand is the exact command ONUSummary runs.
const onuSummaryCommand = "show onu interface all"

// ONUSummary runs "show onu interface all": the OLT's own one-line-per-
// ONU table — operational state, administrative state, service state,
// serial number, registration ID, IP address, MAC address, and
// description — for every ONU on this OLT, across every PON port. It
// returns the device's raw output, verbatim.
func (c *Client) ONUSummary(ctx context.Context) (string, error) {
	return c.run(ctx, onuSummaryCommand)
}

// onuStatusSummaryCommand is the exact command ONUStatusSummary runs.
const onuStatusSummaryCommand = "show onu interface all status"

// ONUStatusSummary runs "show onu interface all status": the same
// one-row-per-ONU shape as ONUSummary, but with operational/
// administrative/ONU state, distance, deactivate reason, and optical
// levels (power measurement time, Rx/Tx power, OLT Rx power) in place of
// ONUSummary's identity fields, for every ONU on this OLT. It returns
// the device's raw output, verbatim.
func (c *Client) ONUStatusSummary(ctx context.Context) (string, error) {
	return c.run(ctx, onuStatusSummaryCommand)
}

// onuRunningConfigTemplate is ONURunningConfig's command, with iface
// standing in for "%s".
const onuRunningConfigTemplate = "show run %s"

// ONURunningConfig runs "show run <iface>": the OLT's stored
// configuration block for that ONU interface — serial number, bound
// service profiles, VLAN/rate settings, description — the same text an
// operator would see confirming how an ONU is currently provisioned. It
// returns the device's raw output, verbatim.
func (c *Client) ONURunningConfig(ctx context.Context, iface string) (string, error) {
	return c.runForInterface(ctx, onuRunningConfigTemplate, iface)
}

// onuDetailTemplate is ONUDetail's command, with iface standing in for
// "%s".
const onuDetailTemplate = "show onu interface %s"

// ONUDetail runs "show onu interface <iface>": the OLT's full detail
// view for one ONU — hardware/firmware identity, capability counts,
// uptime, both firmware image slots, IP/DHCP configuration, MAC address,
// current optical levels and temperature, and service/encryption state.
// This is the single longest output among this package's commands and
// the one confirmed, firsthand, to trigger this platform's pager mid-
// command — see this package's own pager doc comment. It returns the
// device's raw output, verbatim, with any pager prompt already
// transparently stepped past.
func (c *Client) ONUDetail(ctx context.Context, iface string) (string, error) {
	return c.runForInterface(ctx, onuDetailTemplate, iface)
}

// onuStatusTemplate is ONUStatus's command, with iface standing in for
// "%s".
const onuStatusTemplate = "show onu interface %s status"

// ONUStatus runs "show onu interface <iface> status": ONUStatusSummary's
// same status/optical row, for this one ONU alone. It returns the
// device's raw output, verbatim.
func (c *Client) ONUStatus(ctx context.Context, iface string) (string, error) {
	return c.runForInterface(ctx, onuStatusTemplate, iface)
}

// onuEthernetPortsTemplate is ONUEthernetPorts's command, with iface
// standing in for "%s".
const onuEthernetPortsTemplate = "show onu interface %s eth all"

// ONUEthernetPorts runs "show onu interface <iface> eth all": one row
// per physical Ethernet port on that ONU — port type, admin mode, link
// status, speed, duplex, PoE, and DHCP mode. It returns the device's raw
// output, verbatim.
func (c *Client) ONUEthernetPorts(ctx context.Context, iface string) (string, error) {
	return c.runForInterface(ctx, onuEthernetPortsTemplate, iface)
}

// dhcpSnoopingTemplate is DHCPSnoopingEntries's command, with iface
// standing in for "%s".
const dhcpSnoopingTemplate = "show dhcpsnooping interface %s"

// DHCPSnoopingEntries runs "show dhcpsnooping interface <iface>": the
// OLT's DHCP snooping table entries learned on that ONU's ports — IP
// address/prefix, MAC address, VLAN, entry type, and lease time. It
// returns the device's raw output, verbatim.
func (c *Client) DHCPSnoopingEntries(ctx context.Context, iface string) (string, error) {
	return c.runForInterface(ctx, dhcpSnoopingTemplate, iface)
}

// macAddressTableTemplate is MACAddressTableEntries's command, with
// iface standing in for "%s".
const macAddressTableTemplate = "show mac-addr-table interface %s"

// MACAddressTableEntries runs "show mac-addr-table interface <iface>":
// the OLT's learned MAC address table entries for that ONU's ports —
// VLAN, MAC address, interface, and learn status. It returns the
// device's raw output, verbatim.
func (c *Client) MACAddressTableEntries(ctx context.Context, iface string) (string, error) {
	return c.runForInterface(ctx, macAddressTableTemplate, iface)
}
