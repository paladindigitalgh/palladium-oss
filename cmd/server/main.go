// Command server is the entrypoint for the Palladium OSS API. It loads
// configuration, wires dependencies, and runs the HTTP server until an
// interrupt or termination signal triggers a graceful shutdown.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/config"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/health"
	"github.com/paladindigitalgh/palladium-oss/internal/httpserver"
	logging "github.com/paladindigitalgh/palladium-oss/internal/log"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/retry"
	api "github.com/paladindigitalgh/palladium-oss/internal/server"
	"github.com/paladindigitalgh/palladium-oss/internal/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("palladium server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(logger)

	logger.Info("starting palladium server",
		"environment", cfg.Environment,
		"version", version.Version,
		"commit", version.Commit,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbCfg := database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Name,
		SSLMode:         cfg.Database.SSLMode,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
		ConnectTimeout:  cfg.Database.ConnectTimeout,
	}

	pool, err := database.Connect(ctx, dbCfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	warmUpBackoff := retry.NewExponentialBackoff(250*time.Millisecond, 5*time.Second)
	if err := pool.WarmUp(ctx, warmUpBackoff, 5, dbCfg.ConnectTimeout); err != nil {
		logger.Warn("database not reachable yet; continuing startup, readiness will reflect current status",
			"error", err,
		)
	} else {
		logger.Info("database connection verified")
	}

	// Dependency-specific readiness checks are appended here as new
	// dependencies are introduced in later phases.
	healthCheckers := []health.Checker{
		database.NewHealthChecker(pool),
	}

	router := api.NewRouter(api.Dependencies{
		Logger:         logger,
		HealthCheckers: healthCheckers,
		Version:        version.Version,
		Commit:         version.Commit,
	})

	srv := httpserver.New(httpserver.Config{
		Addr:            cfg.Addr(),
		ReadTimeout:     cfg.HTTP.ReadTimeout,
		WriteTimeout:    cfg.HTTP.WriteTimeout,
		IdleTimeout:     cfg.HTTP.IdleTimeout,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}, router, logger)

	if err := srv.Run(ctx); err != nil {
		return err
	}

	logger.Info("palladium server stopped cleanly")
	return nil
}
