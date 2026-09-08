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
// for doing the same.
func (c *Client) run(ctx context.Context, command string) (string, error) {
	out, err := c.shell.RunCommand(ctx, command, diagnosticskontron.Pager)
	if err != nil {
		return "", fmt.Errorf("kontron: %s: %w", command, err)
	}
	return out, nil
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
// in order, over c's shell. Every one of these commands is Cisco-style
// nested config mode (interface drops into a sub-mode where onu
// serial-number and service-profile then run; the two exits climb back
// out, first to config mode, then to the top-level prompt) and each
// takes effect immediately — there is no separate commit step beyond
// save config, which persists the change across a reboot.
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
