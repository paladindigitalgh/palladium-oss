package ssh

import (
	"net"
	"strconv"
	"time"
)

// DefaultPort is used when Config.Port is zero — the standard,
// universally-recognized SSH port, the same default every SSH client
// (OpenSSH included) uses when none is given explicitly.
const DefaultPort = 22

// Config describes how to connect to and authenticate against a single
// SSH server. See this package's doc comment for the full explanation
// of authentication, timeout, and host-key verification behavior; this
// comment covers only what makes a Config valid.
//
// A Config is invalid, and New returns an error without ever opening a
// network connection, if:
//
//   - Host is empty (ErrHostRequired)
//   - Username is empty (ErrUsernameRequired)
//   - both Password and PrivateKey are empty (ErrNoAuthMethod)
//   - StrictHostKeyChecking is true and KnownHostsFile is empty
//     (ErrKnownHostsFileRequired)
//
// Port defaults to DefaultPort and Timeout defaults to DefaultTimeout
// when left zero; neither has a "required" validation error, since both
// have an unambiguous, universally-understood default.
type Config struct {
	// Host is the SSH server's hostname or IP address. Required.
	Host string

	// Port is the SSH server's TCP port. Defaults to DefaultPort (22)
	// when zero.
	Port int

	// Username is the account to authenticate as. Required.
	Username string

	// Password authenticates via SSH's password authentication method.
	// Optional; see this package's doc comment, "Authentication," for
	// how it combines with PrivateKey.
	Password string

	// PrivateKey is a raw, PEM-encoded private key (e.g. the contents of
	// an id_ed25519 file), parsed via golang.org/x/crypto/ssh's
	// ParsePrivateKey. Optional; see this package's doc comment,
	// "Authentication," for how it combines with Password. An
	// encrypted (passphrase-protected) key is not supported.
	PrivateKey []byte

	// KnownHostsFile is the path to an OpenSSH-format known_hosts file,
	// used to verify the server's host key when StrictHostKeyChecking is
	// true. Ignored when StrictHostKeyChecking is false.
	KnownHostsFile string

	// Timeout bounds both connection establishment (New) and each
	// individual command (Run). Defaults to DefaultTimeout (30s) when
	// zero — see this package's doc comment, "Timeout behavior," for why
	// zero does not mean "no timeout."
	Timeout time.Duration

	// StrictHostKeyChecking controls whether the server's host key is
	// verified against KnownHostsFile (true) or accepted unconditionally
	// (false). See this package's doc comment, "Host-key verification,"
	// for why its zero value (false) is a deliberate, security-relevant
	// choice every production caller must explicitly override.
	StrictHostKeyChecking bool
}

// validate reports whether cfg has every required field set, returning
// the first problem found (see Config's own doc comment for the exact
// list). It never touches the network — this is what lets New fail fast
// for a malformed Config without a dial attempt, which is also what lets
// this package's tests exercise every validation failure without a live
// SSH server.
func (cfg Config) validate() error {
	if cfg.Host == "" {
		return ErrHostRequired
	}
	if cfg.Username == "" {
		return ErrUsernameRequired
	}
	if cfg.Password == "" && len(cfg.PrivateKey) == 0 {
		return ErrNoAuthMethod
	}
	if cfg.StrictHostKeyChecking && cfg.KnownHostsFile == "" {
		return ErrKnownHostsFileRequired
	}
	return nil
}

// port returns cfg.Port, or DefaultPort if it is zero.
func (cfg Config) port() int {
	if cfg.Port == 0 {
		return DefaultPort
	}
	return cfg.Port
}

// timeout returns cfg.Timeout, or DefaultTimeout if it is zero.
func (cfg Config) timeout() time.Duration {
	if cfg.Timeout == 0 {
		return DefaultTimeout
	}
	return cfg.Timeout
}

// addr returns cfg.Host and cfg.port() combined into a single
// host:port string suitable for net.Dial and its relatives, via
// net.JoinHostPort rather than a plain Sprintf — a literal IPv6 Host
// (e.g. "::1") needs bracket-aware formatting ("[::1]:22") to parse back
// correctly, which JoinHostPort handles and a naive "%s:%d" would not.
func (cfg Config) addr() string {
	return net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.port()))
}
