package connect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

func validOLT() olt.OLT {
	return olt.OLT{ManagementIPAddress: "192.0.2.10", Vendor: olt.VendorKontron}
}

func validProfile() connectionprofile.ConnectionProfile {
	return connectionprofile.ConnectionProfile{
		Protocol:      "ssh",
		Port:          22,
		Timeout:       5 * time.Second,
		HostKeyPolicy: connectionprofile.HostKeyPolicyInsecure,
	}
}

func passwordAuth() authentication.Authentication {
	return authentication.Authentication{
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "admin",
		Password:           "secret",
	}
}

func TestBuildConfigMapsFieldsForPasswordAuthentication(t *testing.T) {
	cfg, err := buildConfig(validOLT(), validProfile(), passwordAuth(), "")
	if err != nil {
		t.Fatalf("buildConfig() = %v", err)
	}
	if cfg.Host != "192.0.2.10" {
		t.Errorf("Host = %q, want %q", cfg.Host, "192.0.2.10")
	}
	if cfg.Port != 22 {
		t.Errorf("Port = %d, want 22", cfg.Port)
	}
	if cfg.Username != "admin" {
		t.Errorf("Username = %q, want %q", cfg.Username, "admin")
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "secret")
	}
	if len(cfg.PrivateKey) != 0 {
		t.Errorf("PrivateKey = %q, want empty for password auth", cfg.PrivateKey)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", cfg.Timeout)
	}
}

func TestBuildConfigMapsFieldsForSSHKeyAuthentication(t *testing.T) {
	auth := authentication.Authentication{
		AuthenticationType: authentication.AuthenticationTypeSSHKey,
		Username:           "admin",
		PrivateKey:         "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
	}

	cfg, err := buildConfig(validOLT(), validProfile(), auth, "")
	if err != nil {
		t.Fatalf("buildConfig() = %v", err)
	}
	if cfg.Password != "" {
		t.Errorf("Password = %q, want empty for SSH key auth", cfg.Password)
	}
	if string(cfg.PrivateKey) != auth.PrivateKey {
		t.Errorf("PrivateKey = %q, want %q", cfg.PrivateKey, auth.PrivateKey)
	}
}

func TestBuildConfigRejectsEmptyManagementAddress(t *testing.T) {
	target := validOLT()
	target.ManagementIPAddress = ""

	_, err := buildConfig(target, validProfile(), passwordAuth(), "")
	if !errors.Is(err, ErrManagementAddressRequired) {
		t.Fatalf("buildConfig() error = %v, want %v", err, ErrManagementAddressRequired)
	}
}

func TestBuildConfigRejectsUnsupportedAuthenticationType(t *testing.T) {
	auth := passwordAuth()
	auth.AuthenticationType = "Kerberos"

	_, err := buildConfig(validOLT(), validProfile(), auth, "")
	if err == nil {
		t.Fatal("buildConfig() = nil, want an error for an unsupported authentication type")
	}
}

func TestBuildConfigRejectsUnsupportedProtocol(t *testing.T) {
	profile := validProfile()
	profile.Protocol = "telnet"

	_, err := buildConfig(validOLT(), profile, passwordAuth(), "")
	if err == nil {
		t.Fatal("buildConfig() = nil, want an error for an unsupported protocol")
	}
}

// TestBuildConfigAcceptsEmptyOrCaseInsensitiveSSHProtocol proves an
// empty Protocol (a ConnectionProfile that has not had one chosen yet)
// and any case variant of "ssh" are both accepted.
func TestBuildConfigAcceptsEmptyOrCaseInsensitiveSSHProtocol(t *testing.T) {
	for _, protocol := range []string{"", "ssh", "SSH", "Ssh"} {
		t.Run(protocol, func(t *testing.T) {
			profile := validProfile()
			profile.Protocol = protocol

			if _, err := buildConfig(validOLT(), profile, passwordAuth(), ""); err != nil {
				t.Errorf("buildConfig() = %v, want protocol %q accepted", err, protocol)
			}
		})
	}
}

func TestBuildConfigAppliesStrictHostKeyPolicy(t *testing.T) {
	profile := validProfile()
	profile.HostKeyPolicy = connectionprofile.HostKeyPolicyStrict

	cfg, err := buildConfig(validOLT(), profile, passwordAuth(), "/etc/palladium/known_hosts")
	if err != nil {
		t.Fatalf("buildConfig() = %v", err)
	}
	if !cfg.StrictHostKeyChecking {
		t.Error("StrictHostKeyChecking = false, want true for HostKeyPolicyStrict")
	}
	if cfg.KnownHostsFile != "/etc/palladium/known_hosts" {
		t.Errorf("KnownHostsFile = %q, want %q", cfg.KnownHostsFile, "/etc/palladium/known_hosts")
	}
}

func TestBuildConfigAppliesInsecureHostKeyPolicy(t *testing.T) {
	profile := validProfile()
	profile.HostKeyPolicy = connectionprofile.HostKeyPolicyInsecure

	// A known-hosts path supplied anyway (e.g. a caller with one
	// default value for every OLT regardless of policy) must be
	// ignored for an Insecure profile -- only Strict should ever cause
	// ssh.New to attempt to read it.
	cfg, err := buildConfig(validOLT(), profile, passwordAuth(), "/etc/palladium/known_hosts")
	if err != nil {
		t.Fatalf("buildConfig() = %v", err)
	}
	if cfg.StrictHostKeyChecking {
		t.Error("StrictHostKeyChecking = true, want false for HostKeyPolicyInsecure")
	}
	if cfg.KnownHostsFile != "" {
		t.Errorf("KnownHostsFile = %q, want empty for HostKeyPolicyInsecure", cfg.KnownHostsFile)
	}
}

func TestBuildConfigRejectsUnsupportedHostKeyPolicy(t *testing.T) {
	profile := validProfile()
	profile.HostKeyPolicy = "Sometimes"

	_, err := buildConfig(validOLT(), profile, passwordAuth(), "")
	if err == nil {
		t.Fatal("buildConfig() = nil, want an error for an unsupported host key policy")
	}
}

// TestShellWrapsDialFailure proves Shell surfaces a real dial failure
// (rather than hanging or panicking) for an address nothing will ever
// answer on — 192.0.2.0/24 is TEST-NET-1 (RFC 5737), reserved
// specifically for documentation and testing and guaranteed
// non-routable, the same address block internal/platform/ssh's own
// external tests already rely on.
func TestShellWrapsDialFailure(t *testing.T) {
	profile := validProfile()
	profile.Timeout = 200 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Shell(ctx, validOLT(), profile, passwordAuth(), "")
	if err == nil {
		t.Fatal("Shell() = nil error, want a dial failure for an unroutable address")
	}
}

func TestShellPropagatesConfigValidationErrorWithoutDialing(t *testing.T) {
	target := validOLT()
	target.ManagementIPAddress = ""

	_, err := Shell(context.Background(), target, validProfile(), passwordAuth(), "")
	if !errors.Is(err, ErrManagementAddressRequired) {
		t.Fatalf("Shell() error = %v, want %v", err, ErrManagementAddressRequired)
	}
}

// fakeShell and fakeClient below let TestDialedShellCloseClosesBoth
// exercise dialedShell.Close directly, without a real network
// connection.
type fakeShell struct {
	ssh.Shell
	closeErr error
	closed   bool
}

func (f *fakeShell) Close() error {
	f.closed = true
	return f.closeErr
}

type fakeClient struct {
	ssh.Client
	closeErr error
	closed   bool
}

func (f *fakeClient) Close() error {
	f.closed = true
	return f.closeErr
}

// TestDialedShellCloseClosesBoth proves dialedShell.Close closes both
// the shell and the underlying client, and returns the shell's error
// preferentially when both fail -- see dialedShell's own doc comment on
// why leaving the client open matters concretely for a device with a
// small session budget.
func TestDialedShellCloseClosesBoth(t *testing.T) {
	fs := &fakeShell{}
	fc := &fakeClient{}
	d := &dialedShell{Shell: fs, client: fc}

	if err := d.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if !fs.closed {
		t.Error("Close() did not close the shell")
	}
	if !fc.closed {
		t.Error("Close() did not close the underlying client")
	}
}

func TestDialedShellClosePrefersShellError(t *testing.T) {
	shellErr := errors.New("shell close failed")
	fs := &fakeShell{closeErr: shellErr}
	fc := &fakeClient{closeErr: errors.New("client close failed")}
	d := &dialedShell{Shell: fs, client: fc}

	if err := d.Close(); !errors.Is(err, shellErr) {
		t.Fatalf("Close() = %v, want %v", err, shellErr)
	}
	if !fc.closed {
		t.Error("Close() did not still close the underlying client after the shell's own Close failed")
	}
}
