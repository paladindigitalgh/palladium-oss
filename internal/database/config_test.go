package database_test

import (
	"net/url"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
)

func TestDSNIsAValidPostgresURL(t *testing.T) {
	cfg := database.Config{
		Host:     "db.internal",
		Port:     5432,
		User:     "palladium",
		Password: "p@ss/word?",
		Database: "palladium",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("url.Parse(%q) = %v", dsn, err)
	}

	if parsed.Scheme != "postgres" {
		t.Errorf("scheme = %q, want postgres", parsed.Scheme)
	}
	if parsed.Host != "db.internal:5432" {
		t.Errorf("host = %q, want db.internal:5432", parsed.Host)
	}
	if parsed.Path != "/palladium" {
		t.Errorf("path = %q, want /palladium", parsed.Path)
	}
	if got := parsed.User.Username(); got != "palladium" {
		t.Errorf("user = %q, want palladium", got)
	}
	if pw, _ := parsed.User.Password(); pw != "p@ss/word?" {
		t.Errorf("password round-tripped as %q, want p@ss/word?", pw)
	}
	if got := parsed.Query().Get("sslmode"); got != "disable" {
		t.Errorf("sslmode = %q, want disable", got)
	}
}
