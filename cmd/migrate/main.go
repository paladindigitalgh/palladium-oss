// Command migrate applies or inspects Palladium's PostgreSQL schema
// migrations. It is a separate binary from the API server so schema
// changes are an explicit, auditable deploy step rather than something the
// server performs implicitly on boot.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/paladindigitalgh/palladium-oss/database/migrations"
	"github.com/paladindigitalgh/palladium-oss/internal/config"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/database/migrate"
	applog "github.com/paladindigitalgh/palladium-oss/internal/log"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: migrate <up|status>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := applog.New(cfg.Log.Level, cfg.Log.Format)

	dbCfg := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := database.OpenStdlib(dbCfg)
	if err != nil {
		return err
	}
	defer db.Close()

	runner, err := migrate.New(db, migrations.FS)
	if err != nil {
		return err
	}

	ctx := context.Background()

	switch args[0] {
	case "up":
		applied, err := runner.Up(ctx)
		if err != nil {
			return err
		}
		logger.Info("migrations applied", "count", applied)
	case "status":
		version, err := runner.Version(ctx)
		if err != nil {
			return err
		}
		pending, err := runner.HasPending(ctx)
		if err != nil {
			return err
		}
		logger.Info("schema status", "version", version, "pending", pending)
	default:
		return fmt.Errorf("unknown command %q (want up or status)", args[0])
	}

	return nil
}
