package ssh_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// TestNewRejectsInvalidConfigWithoutDialing exercises New — the actual
// public entry point, not the internal newClient client_test.go tests
// against directly — through every validation failure Config
// documents. None of these reach a network dial: an empty Host, an
// empty Username, no auth method, and StrictHostKeyChecking without a
// KnownHostsFile are all caught before New would ever call
// golang.org/x/crypto/ssh.Dial, which is exactly what makes this test
// safe to run without a live SSH server (goal 6) even though it goes
// through the real, non-fake New.
func TestNewRejectsInvalidConfigWithoutDialing(t *testing.T) {
	cases := []struct {
		name    string
		cfg     ssh.Config
		wantErr error
	}{
		{
			name:    "empty config",
			cfg:     ssh.Config{},
			wantErr: ssh.ErrHostRequired,
		},
		{
			name:    "missing username",
			cfg:     ssh.Config{Host: "device.example.test"},
			wantErr: ssh.ErrUsernameRequired,
		},
		{
			name:    "no password or private key",
			cfg:     ssh.Config{Host: "device.example.test", Username: "admin"},
			wantErr: ssh.ErrNoAuthMethod,
		},
		{
			name: "strict host key checking without a known hosts file",
			cfg: ssh.Config{
				Host:                  "device.example.test",
				Username:              "admin",
				Password:              "secret",
				StrictHostKeyChecking: true,
			},
			wantErr: ssh.ErrKnownHostsFileRequired,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client, err := ssh.New(c.cfg)
			if client != nil {
				t.Errorf("New() returned a non-nil Client alongside an error")
			}
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("New() error = %v, want %v", err, c.wantErr)
			}
		})
	}
}

// TestNewAcceptsPasswordOnlyOrPrivateKeyOnlyConfig proves Password and
// PrivateKey are each independently sufficient — "mutually optional," per
// this milestone's exact wording — by confirming neither one alone
// triggers ErrNoAuthMethod. Both of these Configs still fail (there is
// no real "device.example.test" to dial), but they must fail with a
// dial-related error, never ErrNoAuthMethod, proving validation accepted
// the single auth method supplied in each case before attempting to
// connect.
func TestNewAcceptsPasswordOnlyOrPrivateKeyOnlyConfig(t *testing.T) {
	t.Run("password only", func(t *testing.T) {
		_, err := ssh.New(ssh.Config{
			Host: "192.0.2.1", Port: 1, Username: "admin", Password: "secret",
			Timeout: 1,
		})
		if errors.Is(err, ssh.ErrNoAuthMethod) {
			t.Error("New() rejected a Config with Password set as having no auth method")
		}
	})

	t.Run("private key only", func(t *testing.T) {
		_, err := ssh.New(ssh.Config{
			Host: "192.0.2.1", Port: 1, Username: "admin", PrivateKey: []byte("not-a-real-key-but-non-empty"),
			Timeout: 1,
		})
		if errors.Is(err, ssh.ErrNoAuthMethod) {
			t.Error("New() rejected a Config with PrivateKey set as having no auth method")
		}
	})
}

func TestConfigConstants(t *testing.T) {
	if ssh.DefaultPort != 22 {
		t.Errorf("DefaultPort = %d, want 22", ssh.DefaultPort)
	}
	if ssh.DefaultTimeout <= 0 {
		t.Errorf("DefaultTimeout = %v, want a positive duration", ssh.DefaultTimeout)
	}
}
