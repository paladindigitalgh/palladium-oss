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

// Config is the root application configuration.
type Config struct {
	Environment string
	HTTP        HTTPConfig
	Log         LogConfig
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

	return nil
}
