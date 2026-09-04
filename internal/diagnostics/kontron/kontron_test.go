package kontron

import (
	"context"
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// fakeShell is an in-memory ssh.Shell — the same reason
// internal/platform/ssh's own fakes exist: this package's tests exercise
// exactly what Client sends to a Shell and does with what comes back,
// without a live SSH server or a real C16 involved.
type fakeShell struct {
	gotCommand string
	gotPagers  []ssh.PagerPrompt
	output     string
	err        error
	closeErr   error
}

var _ ssh.Shell = (*fakeShell)(nil)

func (f *fakeShell) RunCommand(_ context.Context, command string, pagers ...ssh.PagerPrompt) (string, error) {
	f.gotCommand = command
	f.gotPagers = pagers
	if f.err != nil {
		return "", f.err
	}
	return f.output, nil
}

func (f *fakeShell) Close() error { return f.closeErr }

// onuSummarySample is real "show onu interface all" output captured
// against a production Kontron/Iskratel C16 (a different unit than the
// one used to build internal/platform/ssh's interactive shell support),
// used verbatim so this test proves ONUSummary returns exactly what the
// device printed — no reformatting, no parsing.
const onuSummarySample = `ONU       Oper     Admin    Service   Serial
Interface State    State    State     Number       Password/Registration Id               IP Address      MAC Address       Description
--------- -------- -------- -------- ------------ -------------------------------------- --------------- ----------------- ------------------
xgs/1/1   Up       Enable   Enable   ISKT23089340 ""                                     10.70.178.19    48:55:41:08:93:40 ISKT23089340
xgs/1/2   Up       Enable   Enable   ISKT2308A290 ""                                     10.70.178.21    48:55:41:08:A2:90 ISKT2308A290
xgs/3/7   Down     Enable   Enable   ISKT2308A0C8 ""                                     0.0.0.0         48:55:41:08:A0:C8 ISKT2308A0C8`

func TestONUSummaryRunsExpectedCommandAndReturnsOutputVerbatim(t *testing.T) {
	fs := &fakeShell{output: onuSummarySample}
	c := NewClient(fs)

	got, err := c.ONUSummary(context.Background())
	if err != nil {
		t.Fatalf("ONUSummary() = %v", err)
	}

	if fs.gotCommand != "show onu interface all" {
		t.Errorf("command sent = %q, want %q", fs.gotCommand, "show onu interface all")
	}
	if len(fs.gotPagers) != 1 || fs.gotPagers[0] != pager {
		t.Errorf("pagers sent = %v, want [%v]", fs.gotPagers, pager)
	}
	if got != onuSummarySample {
		t.Errorf("ONUSummary() = %q, want the sample returned verbatim", got)
	}
}

func TestONUSummaryWrapsShellError(t *testing.T) {
	shellErr := errors.New("interactive shell run command: context deadline exceeded")
	fs := &fakeShell{err: shellErr}
	c := NewClient(fs)

	_, err := c.ONUSummary(context.Background())
	if !errors.Is(err, shellErr) {
		t.Fatalf("ONUSummary() error = %v, want it to wrap %v", err, shellErr)
	}
}

// onuDetailSample is real "show onu interface xgs/1/1" output captured
// against a production C16, with the device's own mid-command pager
// prompt already removed — internal/platform/ssh.Shell.RunCommand
// (tested in that package) is what strips it in practice, so a real
// Shell handed to this package would never return it in the first
// place; this sample matches what Client.ONUDetail actually receives.
const onuDetailSample = `------------------------------------------------------------
Interface                               xgs/1/1
Description                             ISKT23089340
Registration ID
Enable PM
Equalization Delay [bit]                1567500
Distance [m]                            4971
Vendor Id                               ISKT
Hardware Version                        InnboxX24_V1.0
Serial Number                           ISKT23089340
Traffic Management Option               onuCfgRateControlledUpstreamTraffic
Operational State                       Up
Administrative State                    Enable
Equipment Id                            InnboxX24
OMCC Version                            g-988-8-2010
Hardware Type                           24
Total Priority Queue Number             152
Total Traffic Scheduler Number          15
Total GEM Port Number                   256
Total TCONT Number                      15
Total Ethernet UNI Number               2
Total POTS UNI Number                   0
System Up Time [DD:HH:MM]               13:3:45
Image Instance 0 Version                1.1.1496
Image Instance 0 Valid                  true
Image Instance 0 Activate               false
Image Instance 0 Commit                 false
Image Instance 1 Version                1.1.1714
Image Instance 1 Valid                  true
Image Instance 1 Activate               true
Image Instance 1 Commit                 true
MAC Address                             48:55:41:08:93:40
DHCP Mode                               true
IP Address                              10.70.178.19
IP Mask                                 255.255.0.0
Default Gateway                         10.70.0.1
Primary DNS                             10.70.0.1
Secondary DNS                           0.0.0.0
MAC Limit                               0
FEC Tx Enable                           true
Upgrade Status                          not applicable
Deactivate Reason                       none
Power measurement start time            21:24:24 28/07/2026
Rx Power [dBm]                          -19.67
Tx Power [dBm]                          6.11
OLT Rx power [dBm]                      -18.54
Temperature (C)                         75.31
Default ONU Configuration File          -
ONU Configuration Status                -
ONU Current Configuration Version       -
Changed Config Attributes               -
Service State                           Enable
ONU Downstream Data Encryption          Enable`

func TestONUDetailRunsExpectedCommandAndReturnsOutputVerbatim(t *testing.T) {
	fs := &fakeShell{output: onuDetailSample}
	c := NewClient(fs)

	got, err := c.ONUDetail(context.Background(), "xgs/1/1")
	if err != nil {
		t.Fatalf("ONUDetail() = %v", err)
	}
	if fs.gotCommand != "show onu interface xgs/1/1" {
		t.Errorf("command sent = %q, want %q", fs.gotCommand, "show onu interface xgs/1/1")
	}
	if got != onuDetailSample {
		t.Errorf("ONUDetail() = %q, want the sample returned verbatim", got)
	}
}

// onuRunningConfigSample is real "show run xgs/1/1" output captured
// against a production C16 — a config stanza, structurally unlike every
// other command's tabular or key/value output, captured here to prove
// this package's "no parsing" guarantee holds regardless of shape.
const onuRunningConfigSample = `interface xgs/1/1
onu serial-number ISKT23089340
service-profile fiberspark500m uni 1 c-vid 1744
service-profile iphost
peak-rate downstream 500032 128
description "ISKT23089340"
exit`

func TestONURunningConfigRunsExpectedCommandAndReturnsOutputVerbatim(t *testing.T) {
	fs := &fakeShell{output: onuRunningConfigSample}
	c := NewClient(fs)

	got, err := c.ONURunningConfig(context.Background(), "xgs/1/1")
	if err != nil {
		t.Fatalf("ONURunningConfig() = %v", err)
	}
	if fs.gotCommand != "show run xgs/1/1" {
		t.Errorf("command sent = %q, want %q", fs.gotCommand, "show run xgs/1/1")
	}
	if got != onuRunningConfigSample {
		t.Errorf("ONURunningConfig() = %q, want the sample returned verbatim", got)
	}
}

// TestClientMethodsSendExpectedCommands covers every remaining Client
// method's command string and pager wiring in one table, now that
// ONUSummary, ONUDetail, and ONURunningConfig above have each already
// proven the same plumbing works end to end against real device output.
func TestClientMethodsSendExpectedCommands(t *testing.T) {
	const iface = "xgs/1/1"

	cases := []struct {
		name        string
		call        func(c *Client, ctx context.Context) (string, error)
		wantCommand string
	}{
		{"ONUSummary", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUSummary(ctx)
		}, "show onu interface all"},
		{"ONUStatusSummary", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUStatusSummary(ctx)
		}, "show onu interface all status"},
		{"ONURunningConfig", func(c *Client, ctx context.Context) (string, error) {
			return c.ONURunningConfig(ctx, iface)
		}, "show run xgs/1/1"},
		{"ONUDetail", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUDetail(ctx, iface)
		}, "show onu interface xgs/1/1"},
		{"ONUStatus", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUStatus(ctx, iface)
		}, "show onu interface xgs/1/1 status"},
		{"ONUEthernetPorts", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUEthernetPorts(ctx, iface)
		}, "show onu interface xgs/1/1 eth all"},
		{"DHCPSnoopingEntries", func(c *Client, ctx context.Context) (string, error) {
			return c.DHCPSnoopingEntries(ctx, iface)
		}, "show dhcpsnooping interface xgs/1/1"},
		{"MACAddressTableEntries", func(c *Client, ctx context.Context) (string, error) {
			return c.MACAddressTableEntries(ctx, iface)
		}, "show mac-addr-table interface xgs/1/1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := &fakeShell{output: "sample output"}
			c := NewClient(fs)

			got, err := tc.call(c, context.Background())
			if err != nil {
				t.Fatalf("%s() = %v", tc.name, err)
			}
			if fs.gotCommand != tc.wantCommand {
				t.Errorf("command sent = %q, want %q", fs.gotCommand, tc.wantCommand)
			}
			if len(fs.gotPagers) != 1 || fs.gotPagers[0] != pager {
				t.Errorf("pagers sent = %v, want [%v]", fs.gotPagers, pager)
			}
			if got != "sample output" {
				t.Errorf("%s() = %q, want the sample returned verbatim", tc.name, got)
			}
		})
	}
}

// TestPerInterfaceMethodsRejectEmbeddedNewline proves every per-
// interface method rejects an iface containing a newline before ever
// reaching the shell — see ErrInvalidInterface's own doc comment for
// why this is a command-injection guard, not format validation.
func TestPerInterfaceMethodsRejectEmbeddedNewline(t *testing.T) {
	const malicious = "xgs/1/1\nreload"

	cases := []struct {
		name string
		call func(c *Client, ctx context.Context) (string, error)
	}{
		{"ONURunningConfig", func(c *Client, ctx context.Context) (string, error) {
			return c.ONURunningConfig(ctx, malicious)
		}},
		{"ONUDetail", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUDetail(ctx, malicious)
		}},
		{"ONUStatus", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUStatus(ctx, malicious)
		}},
		{"ONUEthernetPorts", func(c *Client, ctx context.Context) (string, error) {
			return c.ONUEthernetPorts(ctx, malicious)
		}},
		{"DHCPSnoopingEntries", func(c *Client, ctx context.Context) (string, error) {
			return c.DHCPSnoopingEntries(ctx, malicious)
		}},
		{"MACAddressTableEntries", func(c *Client, ctx context.Context) (string, error) {
			return c.MACAddressTableEntries(ctx, malicious)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := &fakeShell{output: "should never be reached"}
			c := NewClient(fs)

			_, err := tc.call(c, context.Background())
			if !errors.Is(err, ErrInvalidInterface) {
				t.Fatalf("%s() error = %v, want %v", tc.name, err, ErrInvalidInterface)
			}
			if fs.gotCommand != "" {
				t.Errorf("%s(): shell.RunCommand was called with %q; it must never be reached for an invalid interface", tc.name, fs.gotCommand)
			}
		})
	}
}
