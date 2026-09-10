// Package kontron implements Palladium's first real vendor-specific
// *write* command surface: the config-change CLI sequence a
// Kontron/Iskratel C16 OLT needs to authorize a physically-detected but
// unauthorized ONU (see internal/diagnostics/kontron.BlacklistedONUs) so
// it can be brought into service. Every command here was confirmed
// firsthand by the person operating this equipment, the same standard
// internal/diagnostics/kontron holds itself to for its own read-only
// commands.
//
// This package is not a replacement for, or refactor of, the existing
// internal/provisioning (ProvisioningProfile — the Product-to-vendor-
// profile-name mapping catalog): that package is pure reference data,
// explicitly scoped to never execute a command or open a connection
// (see its own doc comment). This package is that reference data's
// eventual live-action counterpart — the thing that actually runs
// commands against a device — landing here, as a sibling, because it is
// the same overall responsibility ("provisioning") at a different,
// later stage of that responsibility's own build-out, not a competing
// implementation of it.
//
// This package depends on internal/diagnostics/kontron (its
// ONUSummary command, ParseONUSummaryEntries, and Pager — see
// NextFreeIndex and AuthorizeONU below) to read an OLT's current ONU
// assignments before deciding where to place a new one. That dependency
// is one-way and load-bearing: internal/diagnostics/kontron must never
// import this package back. The entire value of a "read-only diagnostics"
// package is that something can depend on it without thereby gaining the
// ability to change device configuration — that guarantee breaks the
// moment the dependency runs in the other direction.
package kontron

import (
	"context"
	"errors"
	"fmt"
	"strings"

	diagnosticskontron "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// ErrInvalidInterface means an iface argument passed to AuthorizeONU
// contained a newline or carriage return. See
// internal/diagnostics/kontron.ErrInvalidInterface's own doc comment for
// the full reasoning — it applies identically here: iface is
// interpolated directly into a command line sent to an interactive
// shell, where a stray newline is a command-injection shape, not a
// formatting nitpick.
var ErrInvalidInterface = errors.New("kontron: interface value contains a newline")

// ErrInvalidSerialNumber is ErrInvalidInterface's counterpart for
// AuthorizeONU's serialNumber argument — the same command-injection
// shape, on the other argument this package interpolates into a command
// line.
var ErrInvalidSerialNumber = errors.New("kontron: serial number value contains a newline")

// ErrInvalidManagementServiceProfile is ErrInvalidInterface's counterpart
// for AuthorizeONU's managementServiceProfile argument — the same
// command-injection shape, on the third and last argument this package
// interpolates into a command line.
var ErrInvalidManagementServiceProfile = errors.New("kontron: management service profile value contains a newline")

// ErrInvalidProfileName is ErrInvalidInterface's counterpart for
// ApplyServiceProfile's profileName argument — the same command-
// injection shape, on the second and last argument that method
// interpolates into a command line.
var ErrInvalidProfileName = errors.New("kontron: profile name value contains a newline")

// Client runs the Kontron/Iskratel C16 ONU-authorization command
// sequence over an already-open interactive shell.
//
// Like internal/diagnostics/kontron.Client, whose shape this
// deliberately mirrors, a Client is handed an already-open ssh.Shell and
// does not take ownership of its lifecycle — the caller remains
// responsible for closing it.
type Client struct {
	shell ssh.Shell
}

// NewClient builds a Client around shell, which must already be open
// and logged in to the target OLT.
func NewClient(shell ssh.Shell) *Client {
	return &Client{shell: shell}
}

// run executes command and wraps any failure with the command itself, so
// an error surfaced further up the stack says what was actually run —
// the same reasoning internal/diagnostics/kontron.Client's own run gives
// for doing the same. The returned string has command's own echo
// stripped from its front (see stripEcho) before every caller in this
// file checks it against the empty string for the "did this step
// succeed" convention each of them documents.
func (c *Client) run(ctx context.Context, command string) (string, error) {
	out, err := c.shell.RunCommand(ctx, command, diagnosticskontron.Pager)
	if err != nil {
		return "", fmt.Errorf("kontron: %s: %w", command, err)
	}
	return stripEcho(out, command), nil
}

// stripEcho removes command's own echo from the front of out, if
// present. Confirmed firsthand against a real Kontron/Iskratel C16: raw
// output for a genuinely successful command is not actually empty, it
// is exactly the device's own echo of what was sent (e.g. "configure"
// itself produced the raw string "configure\n\r") — internal/platform/
// ssh.Shell's own doc comment already documents that a device may echo
// typed input regardless of the requested PTY's ECHO mode, but this
// package's own "empty output means success" convention did not
// account for it until this was caught against real hardware, since
// every existing test's fake Shell never modeled this echo happening
// at all. A leading '\r' or '\n' before the echo — the same PTY prompt-
// redraw artifact internal/platform/ssh's detectInitialPrompt already
// tolerates — is stripped first, so the remaining, real device response
// (if any) is what every caller's emptiness check actually sees.
func stripEcho(out, command string) string {
	return strings.TrimPrefix(strings.TrimLeft(out, "\r\n"), command)
}

// AuthorizeONU runs the seven-command sequence confirmed by the person
// operating this equipment:
//
//	configure
//	interface <iface>
//	onu serial-number <serialNumber>
//	service-profile <managementServiceProfile>
//	exit
//	exit
//	save config
//
// in order, over c's shell. configure requires privileged EXEC
// ("HOSTNAME#") — confirmed firsthand against a real Kontron/Iskratel
// C16 that the account this connects as lands in unprivileged user EXEC
// ("HOSTNAME>") when authenticated via the plain "password" SSH method,
// where configure fails with "% Invalid input detected", but in
// privileged EXEC when authenticated via keyboard-interactive instead —
// a quirk of this device's own AAA, not anything about the account
// itself (see internal/platform/ssh's client construction, which tries
// keyboard-interactive first for exactly this reason). No "enable" step
// exists to elevate privilege after connecting: it is not a valid
// command at all on this device's CLI, confirmed firsthand.
//
// Every command from configure onward is Cisco-style nested config mode
// (interface drops into a sub-mode where onu serial-number and
// service-profile then run; the two exits climb back out, first to
// config mode, then to the top-level prompt) and each takes effect
// immediately — there is no separate commit step beyond save config,
// which persists the change across a reboot. Entering and leaving these
// modes changes the device's prompt text ("HOSTNAME#" <->
// "HOSTNAME(Config)#"), which internal/platform/ssh/interactive.go's
// promptPattern is what recognizes as "the prompt is back" — a literal-
// string match cannot, since the mode suffix differs from whatever
// prompt was first detected at connection time.
//
// managementServiceProfile is always applied, on every ONU, unlike a
// subscriber's actual service profile (a separate, later step that
// varies by whatever product/plan the customer bought): it wires the
// ONU's own management interface onto the management network so it can
// get a DHCP address and receive firmware/maintenance from the operator,
// and it is meant to stay on the ONU for its entire lifecycle, never
// removed or changed by a service change. It is a caller-supplied
// argument, not a hardcoded literal, because — unlike every other string
// in this sequence — it names a service profile a specific operator
// configured on their own OLTs (defaulting to "iphost"; see
// internal/config's KontronConfig), not a fixed fact about the Kontron
// CLI itself.
//
// Confirmed success looks like every one of these commands returning
// silently to the prompt (empty output). The only known failure mode is
// a device-reported message on the onu serial-number line (e.g.
// "onu_serial_number wrong size!" for a serial number that is not 12
// characters), but that is not confirmed to be the only possible failure
// on any of these lines — so this is handled generically, not by
// matching specific known strings: the first step whose output is
// non-empty aborts the sequence immediately (no further commands are
// run), and that raw text becomes the returned error.
func (c *Client) AuthorizeONU(ctx context.Context, iface, serialNumber, managementServiceProfile string) error {
	if strings.ContainsAny(iface, "\n\r") {
		return ErrInvalidInterface
	}
	if strings.ContainsAny(serialNumber, "\n\r") {
		return ErrInvalidSerialNumber
	}
	if strings.ContainsAny(managementServiceProfile, "\n\r") {
		return ErrInvalidManagementServiceProfile
	}

	steps := []string{
		"configure",
		fmt.Sprintf("interface %s", iface),
		fmt.Sprintf("onu serial-number %s", serialNumber),
		fmt.Sprintf("service-profile %s", managementServiceProfile),
		"exit",
		"exit",
		"save config",
	}
	for _, step := range steps {
		out, err := c.run(ctx, step)
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("kontron: %s: %s", step, strings.TrimSpace(out))
		}
	}
	return nil
}

// RemoveServiceProfile runs the six-command sequence confirmed by the
// person operating this equipment for removing a subscriber's service
// profile from an already-authorized ONU — the mirror image of
// ApplyServiceProfile:
//
//	configure
//	interface <iface>
//	no service-profile <profileName>
//	exit
//	exit
//	save config
//
// Deliberately no "uni <N>" suffix here, unlike ApplyServiceProfile,
// confirmed live against the real Kontron/Iskratel C16 (2026-09-10): the
// device rejects "no service-profile <profileName> uni <N>" outright
// ("Invalid input detected at '^' marker") — removal is unqualified by
// LAN port, unlike application. An earlier version of this method
// accepted a uniPort argument to mirror ApplyServiceProfile's signature;
// that was wrong and has been removed, not just unused, so a future
// reader never assumes symmetry between apply and remove that the real
// hardware does not have.
//
// This is used for both Suspend and Disconnect: at the Kontron config
// level they are the identical action (take the named profile back
// off), the difference between them living entirely in
// internal/workflow/engine's own mapping of Capability to the Service's
// resulting lifecycle status, not in anything this package does
// differently. The ONU's own authorization (see AuthorizeONU) and its
// management service-profile are untouched — only the profile named by
// profileName is removed.
//
// Success/failure detection follows ApplyServiceProfile's exact
// convention: the first step whose output is non-empty aborts the
// sequence and becomes the returned error.
func (c *Client) RemoveServiceProfile(ctx context.Context, iface, profileName string) error {
	if strings.ContainsAny(iface, "\n\r") {
		return ErrInvalidInterface
	}
	if strings.ContainsAny(profileName, "\n\r") {
		return ErrInvalidProfileName
	}

	steps := []string{
		"configure",
		fmt.Sprintf("interface %s", iface),
		fmt.Sprintf("no service-profile %s", profileName),
		"exit",
		"exit",
		"save config",
	}
	for _, step := range steps {
		out, err := c.run(ctx, step)
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("kontron: %s: %s", step, strings.TrimSpace(out))
		}
	}
	return nil
}

// ApplyServiceProfile runs the six-command sequence confirmed by the
// person operating this equipment for applying a subscriber's own
// service profile to an already-authorized ONU:
//
//	configure
//	interface <iface>
//	service-profile <profileName> uni <uniPort>
//	exit
//	exit
//	save config
//
// in order, over c's shell. Unlike AuthorizeONU, this does not run "onu
// serial-number" — the ONU on iface is assumed already authorized (see
// AuthorizeONU), and this stacks a second, product-specific
// service-profile on top of the management one AuthorizeONU already
// applied, in the same interface context. profileName names a
// ProvisioningProfile.ProfileName (see internal/provisioning) — the
// vendor-side profile a specific commercial Product maps to, not a fixed
// literal this package owns. uniPort selects which of the ONU's two LAN
// ports the profile binds to — 1 for "10GE", 2 for "1GE" — an
// operator-chosen value at Add Service time (see
// internal/serviceequipment.ServiceEquipment's own doc comment); this
// package trusts it is already 1 or 2 (see
// ServiceEquipment.Validate) rather than re-validating it here, the same
// "validate once, at the domain boundary" convention this codebase
// follows throughout.
//
// Success/failure detection follows AuthorizeONU's exact convention: the
// first step whose output is non-empty aborts the sequence and becomes
// the returned error.
func (c *Client) ApplyServiceProfile(ctx context.Context, iface, profileName string, uniPort int) error {
	if strings.ContainsAny(iface, "\n\r") {
		return ErrInvalidInterface
	}
	if strings.ContainsAny(profileName, "\n\r") {
		return ErrInvalidProfileName
	}

	steps := []string{
		"configure",
		fmt.Sprintf("interface %s", iface),
		fmt.Sprintf("service-profile %s uni %d", profileName, uniPort),
		"exit",
		"exit",
		"save config",
	}
	for _, step := range steps {
		out, err := c.run(ctx, step)
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("kontron: %s: %s", step, strings.TrimSpace(out))
		}
	}
	return nil
}

// DeauthorizeONU runs the seven-command sequence confirmed by the person
// operating this equipment for fully removing an ONU's authorization
// from an interface — the mirror image of AuthorizeONU, not of
// ApplyServiceProfile/RemoveServiceProfile: those two leave the ONU's
// base authorization and management service-profile in place, changing
// only the subscriber-specific profile on top of it, so that a
// suspended or disconnected ONU stays reachable on the management
// network. DeauthorizeONU removes that base authorization entirely —
// for decommissioning, swapping, or otherwise fully pulling a device
// out of service:
//
//	configure
//	interface <iface>
//	no service-profile <managementServiceProfile>
//	no onu serial-number
//	exit
//	exit
//	save config
//
// "no service-profile <managementServiceProfile>" runs first, undoing
// AuthorizeONU's own two authorization steps in reverse order: the OLT
// would not fully release the ONU's serial number for "no onu
// serial-number" alone to clear it while the management service-profile
// AuthorizeONU applied is still bound to it — a real failure this
// package's caller hit in practice (attempting to re-authorize a serial
// number this method had supposedly already deauthorized failed with
// "Serial number already exists", even though the ONU no longer appeared
// in "show onu interface all"), traced back to this step's absence and
// confirmed fixed by the same person operating this equipment who
// confirmed the original sequence.
//
// Unlike AuthorizeONU, no serial number is supplied here: "no onu
// serial-number" takes no argument, clearing whatever is currently
// authorized on iface. Success/failure detection follows every other
// Client method's exact convention: the first step whose output is
// non-empty aborts the sequence and becomes the returned error.
func (c *Client) DeauthorizeONU(ctx context.Context, iface, managementServiceProfile string) error {
	if strings.ContainsAny(iface, "\n\r") {
		return ErrInvalidInterface
	}
	if strings.ContainsAny(managementServiceProfile, "\n\r") {
		return ErrInvalidManagementServiceProfile
	}

	steps := []string{
		"configure",
		fmt.Sprintf("interface %s", iface),
		fmt.Sprintf("no service-profile %s", managementServiceProfile),
		"no onu serial-number",
		"exit",
		"exit",
		"save config",
	}
	for _, step := range steps {
		out, err := c.run(ctx, step)
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("kontron: %s: %s", step, strings.TrimSpace(out))
		}
	}
	return nil
}
