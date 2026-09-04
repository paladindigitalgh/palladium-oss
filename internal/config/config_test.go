package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Addr() != "0.0.0.0:8080" {
		t.Errorf("Addr() = %q, want %q", cfg.Addr(), "0.0.0.0:8080")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, "json")
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want 5432", cfg.Database.Port)
	}
	if cfg.Database.MaxConns < cfg.Database.MinConns {
		t.Errorf("Database.MaxConns (%d) < Database.MinConns (%d)", cfg.Database.MaxConns, cfg.Database.MinConns)
	}
	if cfg.JWT.Secret == "" {
		t.Error("JWT.Secret = \"\", want a non-empty dev default")
	}
	if cfg.JWT.Expiration <= 0 {
		t.Errorf("JWT.Expiration = %v, want > 0", cfg.JWT.Expiration)
	}
	if cfg.Encryption.MasterKey == "" {
		t.Error("Encryption.MasterKey = \"\", want a non-empty dev default")
	}
	if cfg.SSH.KnownHostsFile != "" {
		t.Errorf("SSH.KnownHostsFile = %q, want empty by default (no placeholder value makes sense here)", cfg.SSH.KnownHostsFile)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("HTTP_HOST", "127.0.0.1")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("SSH_KNOWN_HOSTS_FILE", "/etc/palladium/known_hosts")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Addr() != "127.0.0.1:9090" {
		t.Errorf("Addr() = %q, want %q", cfg.Addr(), "127.0.0.1:9090")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want %q", cfg.Log.Format, "text")
	}
	if cfg.SSH.KnownHostsFile != "/etc/palladium/known_hosts" {
		t.Errorf("SSH.KnownHostsFile = %q, want %q", cfg.SSH.KnownHostsFile, "/etc/palladium/known_hosts")
	}
}

func TestValidateRejectsBadPort(t *testing.T) {
	cfg := Config{HTTP: HTTPConfig{Port: 0}, Log: LogConfig{Level: "info", Format: "json"}}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for port 0")
	}
}

func TestValidateRejectsBadLogLevel(t *testing.T) {
	cfg := Config{HTTP: HTTPConfig{Port: 8080}, Log: LogConfig{Level: "verbose", Format: "json"}}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for invalid log level")
	}
}

func TestValidateRejectsBadLogFormat(t *testing.T) {
	cfg := Config{HTTP: HTTPConfig{Port: 8080}, Log: LogConfig{Level: "info", Format: "xml"}}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for invalid log format")
	}
}

func TestValidateRejectsBadDBPort(t *testing.T) {
	cfg := Config{
		HTTP:     HTTPConfig{Port: 8080},
		Log:      LogConfig{Level: "info", Format: "json"},
		Database: DatabaseConfig{Port: 0, Name: "palladium", MaxConns: 10, MinConns: 2},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for invalid DB_PORT")
	}
}

func TestValidateRejectsMinConnsAboveMaxConns(t *testing.T) {
	cfg := Config{
		HTTP:     HTTPConfig{Port: 8080},
		Log:      LogConfig{Level: "info", Format: "json"},
		Database: DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 2, MinConns: 5},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error when DB_MIN_CONNS exceeds DB_MAX_CONNS")
	}
}

func TestValidateRejectsEmptyJWTSecret(t *testing.T) {
	cfg := Config{
		HTTP:     HTTPConfig{Port: 8080},
		Log:      LogConfig{Level: "info", Format: "json"},
		Database: DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 10, MinConns: 2},
		JWT:      JWTConfig{Secret: "", Expiration: time.Hour},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for empty JWT_SECRET")
	}
}

func TestValidateRejectsDefaultJWTSecretInProduction(t *testing.T) {
	cfg := Config{
		Environment: "production",
		HTTP:        HTTPConfig{Port: 8080},
		Log:         LogConfig{Level: "info", Format: "json"},
		Database:    DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 10, MinConns: 2},
		JWT:         JWTConfig{Secret: defaultJWTSecret, Expiration: time.Hour},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for the dev-default JWT_SECRET in production")
	}
}

func TestValidateAllowsDefaultJWTSecretOutsideProduction(t *testing.T) {
	cfg := Config{
		Environment: "development",
		HTTP:        HTTPConfig{Port: 8080},
		Log:         LogConfig{Level: "info", Format: "json"},
		Database:    DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 10, MinConns: 2},
		JWT:         JWTConfig{Secret: defaultJWTSecret, Expiration: time.Hour},
		Encryption:  EncryptionConfig{MasterKey: defaultMasterKey},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for the dev-default JWT_SECRET outside production", err)
	}
}

func TestValidateRejectsEmptyMasterKey(t *testing.T) {
	cfg := Config{
		HTTP:       HTTPConfig{Port: 8080},
		Log:        LogConfig{Level: "info", Format: "json"},
		Database:   DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 10, MinConns: 2},
		JWT:        JWTConfig{Secret: "s", Expiration: time.Hour},
		Encryption: EncryptionConfig{MasterKey: ""},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for empty PALLADIUM_MASTER_KEY")
	}
}

func TestValidateRejectsDefaultMasterKeyInProduction(t *testing.T) {
	cfg := Config{
		Environment: "production",
		HTTP:        HTTPConfig{Port: 8080},
		Log:         LogConfig{Level: "info", Format: "json"},
		Database:    DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 10, MinConns: 2},
		JWT:         JWTConfig{Secret: "s", Expiration: time.Hour},
		Encryption:  EncryptionConfig{MasterKey: defaultMasterKey},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for the dev-default PALLADIUM_MASTER_KEY in production")
	}
}

func TestValidateAllowsDefaultMasterKeyOutsideProduction(t *testing.T) {
	cfg := Config{
		Environment: "development",
		HTTP:        HTTPConfig{Port: 8080},
		Log:         LogConfig{Level: "info", Format: "json"},
		Database:    DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 10, MinConns: 2},
		JWT:         JWTConfig{Secret: "s", Expiration: time.Hour},
		Encryption:  EncryptionConfig{MasterKey: defaultMasterKey},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for the dev-default PALLADIUM_MASTER_KEY outside production", err)
	}
}

func TestValidateRejectsNonPositiveJWTExpiration(t *testing.T) {
	cfg := Config{
		HTTP:     HTTPConfig{Port: 8080},
		Log:      LogConfig{Level: "info", Format: "json"},
		Database: DatabaseConfig{Port: 5432, Name: "palladium", MaxConns: 10, MinConns: 2},
		JWT:      JWTConfig{Secret: "s", Expiration: 0},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for non-positive JWT_EXPIRATION")
	}
}

func TestGetEnvIntFallsBackOnInvalidValue(t *testing.T) {
	t.Setenv("TEST_INT_VALUE", "not-a-number")
	if got := getEnvInt("TEST_INT_VALUE", 42); got != 42 {
		t.Errorf("getEnvInt() = %d, want fallback 42", got)
	}
}

func TestGetEnvDurationFallsBackOnInvalidValue(t *testing.T) {
	t.Setenv("TEST_DURATION_VALUE", "not-a-duration")
	if got := getEnvDuration("TEST_DURATION_VALUE", 0); got != 0 {
		t.Errorf("getEnvDuration() = %v, want fallback 0", got)
	}
}
