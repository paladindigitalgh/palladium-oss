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

	return nil
}
