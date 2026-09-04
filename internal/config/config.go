// Package config loads and validates application configuration from
// environment variables.
package config

import (
	"fmt"
	"time"
)

// HTTPConfig holds settings for the HTTP server.
type HTTPConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	// AllowedOrigin is the one frontend origin the API accepts
	// cross-origin requests from (see internal/server's CORS
	// middleware). The frontend and API run on different ports in
	// development (Vite's 5173 vs the API's 8080), so without this a
	// browser rejects every request before it ever reaches a handler --
	// this is not optional dev polish, it is required for the frontend
	// to work in a real browser at all.
	AllowedOrigin string
}

// LogConfig holds settings for structured logging.
type LogConfig struct {
	Level  string
	Format string
}

// defaultJWTSecret is used when JWT_SECRET is unset. It exists purely for
// local development convenience (see configs/.env.example), matching how
// DatabaseConfig.Password already defaults to a known dev value — but
// unlike a leaked dev DB password, a leaked JWT secret lets an attacker
// forge a valid token for any user, so Validate additionally refuses to
// start with this exact value when APP_ENV=production, rather than only
// checking for an empty string.
const defaultJWTSecret = "development-only-change-me-before-deploying"

// JWTConfig holds settings for signing and validating authentication JWTs
// (see internal/auth.TokenIssuer).
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// defaultMasterKey is used when PALLADIUM_MASTER_KEY is unset — a
// pre-generated, valid base64-encoded 32-byte key, existing purely for
// local development convenience, exactly mirroring defaultJWTSecret's
// own reasoning and the same production-override enforcement in
// Validate. A leaked default encryption key is arguably worse than a
// leaked default JWT secret: it does not just let an attacker forge
// tokens, it lets them decrypt every Authentication record's Password
// and PrivateKey ever encrypted under it (see internal/authentication
// and internal/platform/encryption).
const defaultMasterKey = "80azDk64Qom7+ANSVtnDef+mNNinbm98+Da6TsIucH8="

// EncryptionConfig holds settings for encrypting secrets at rest (see
// internal/platform/encryption).
type EncryptionConfig struct {
	MasterKey string
}

// SSHConfig holds settings for connecting to network devices over SSH
// (see internal/platform/ssh and internal/olt/connect).
type SSHConfig struct {
	// KnownHostsFile is the OpenSSH-format known_hosts file used to
	// verify a device's host key for any connection whose
	// ConnectionProfile specifies
	// connectionprofile.HostKeyPolicyStrict — see
	// internal/olt/connect.Shell's own doc comment on knownHostsFile.
	// It is unused, and may be left empty, for every ConnectionProfile
	// using HostKeyPolicyInsecure instead.
	//
	// There is deliberately no default value here (unlike
	// defaultJWTSecret and defaultMasterKey above): unlike a secret,
	// there is no dev-convenience placeholder for a host verification
	// file that would mean anything — an empty value simply means no
	// Strict-policy connection can succeed yet, which
	// internal/platform/ssh.Config.validate already reports clearly
	// (ErrKnownHostsFileRequired) the moment one is attempted.
	KnownHostsFile string
}

// DatabaseConfig holds settings for the PostgreSQL connection pool.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

// Config is the root application configuration.
type Config struct {
	Environment string
	HTTP        HTTPConfig
	Log         LogConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	Encryption  EncryptionConfig
	SSH         SSHConfig
}

// Load builds a Config from environment variables, applying defaults for
// anything left unset, and validates the result.
func Load() (Config, error) {
	cfg := Config{
		Environment: getEnvString("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Host:            getEnvString("HTTP_HOST", "0.0.0.0"),
			Port:            getEnvInt("HTTP_PORT", 8080),
			ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     getEnvDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			AllowedOrigin:   getEnvString("CORS_ALLOWED_ORIGIN", "http://localhost:5173"),
		},
		Log: LogConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
		Database: DatabaseConfig{
			Host:            getEnvString("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnvString("DB_USER", "palladium"),
			Password:        getEnvString("DB_PASSWORD", "palladium"),
			Name:            getEnvString("DB_NAME", "palladium"),
			SSLMode:         getEnvString("DB_SSLMODE", "disable"),
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 10)),
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
			MaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
			ConnectTimeout:  getEnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
		},
		JWT: JWTConfig{
			Secret: getEnvString("JWT_SECRET", defaultJWTSecret),
			// There are no refresh tokens (out of scope for this
			// milestone), so this duration alone determines how often an
			// authenticated caller must log in again. 24h is a pragmatic
			// default for a shift-based operational tool, not a security
			// requirement — operators should tune JWT_EXPIRATION to their
			// own posture.
			Expiration: getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
		},
		Encryption: EncryptionConfig{
			MasterKey: getEnvString("PALLADIUM_MASTER_KEY", defaultMasterKey),
		},
		SSH: SSHConfig{
			KnownHostsFile: getEnvString("SSH_KNOWN_HOSTS_FILE", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Addr returns the host:port pair the HTTP server should bind to.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.HTTP.Host, c.HTTP.Port)
}

// Validate reports whether the configuration is usable.
func (c Config) Validate() error {
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		return fmt.Errorf("config: invalid HTTP_PORT %d", c.HTTP.Port)
	}

	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("config: invalid LOG_LEVEL %q", c.Log.Level)
	}

	switch c.Log.Format {
	case "json", "text":
	default:
		return fmt.Errorf("config: invalid LOG_FORMAT %q", c.Log.Format)
	}

	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("config: invalid DB_PORT %d", c.Database.Port)
	}
	if c.Database.Name == "" {
		return fmt.Errorf("config: DB_NAME must not be empty")
	}
	if c.Database.MaxConns < 1 {
		return fmt.Errorf("config: DB_MAX_CONNS must be at least 1")
	}
	if c.Database.MinConns < 0 || c.Database.MinConns > c.Database.MaxConns {
		return fmt.Errorf("config: DB_MIN_CONNS must be between 0 and DB_MAX_CONNS")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("config: JWT_SECRET must not be empty")
	}
	if c.Environment == "production" && c.JWT.Secret == defaultJWTSecret {
		return fmt.Errorf("config: JWT_SECRET must be overridden in production")
	}
	if c.JWT.Expiration <= 0 {
		return fmt.Errorf("config: JWT_EXPIRATION must be positive")
	}

	if c.Encryption.MasterKey == "" {
		return fmt.Errorf("config: PALLADIUM_MASTER_KEY must not be empty")
	}
	if c.Environment == "production" && c.Encryption.MasterKey == defaultMasterKey {
		return fmt.Errorf("config: PALLADIUM_MASTER_KEY must be overridden in production")
	}

	return nil
}
