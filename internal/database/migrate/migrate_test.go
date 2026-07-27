//go:build integration

package migrate_test

import (
	"context"
	"os"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/database/migrations"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/database/migrate"
)

func TestUpAppliesBootstrapMigration(t *testing.T) {
	cfg := database.Config{
		Host:     envOrDefault("DB_HOST", "localhost"),
		Port:     5432,
		User:     envOrDefault("DB_USER", "palladium"),
		Password: envOrDefault("DB_PASSWORD", "palladium"),
		Database: envOrDefault("DB_NAME", "palladium"),
		SSLMode:  "disable",
	}

	db, err := database.OpenStdlib(cfg)
	if err != nil {
		t.Fatalf("OpenStdlib() = %v", err)
	}
	defer db.Close()

	runner, err := migrate.New(db, migrations.FS)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	ctx := context.Background()

	if _, err := runner.Up(ctx); err != nil {
		t.Fatalf("Up() = %v; is Postgres running? try `make db-up`", err)
	}

	pending, err := runner.HasPending(ctx)
	if err != nil {
		t.Fatalf("HasPending() = %v", err)
	}
	if pending {
		t.Error("HasPending() = true after Up(), want false")
	}

	version, err := runner.Version(ctx)
	if err != nil {
		t.Fatalf("Version() = %v", err)
	}
	if version < 1 {
		t.Errorf("Version() = %d, want >= 1", version)
	}

	var extensionExists bool
	row := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto')")
	if err := row.Scan(&extensionExists); err != nil {
		t.Fatalf("query pg_extension: %v", err)
	}
	if !extensionExists {
		t.Error("pgcrypto extension was not created by the bootstrap migration")
	}
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
