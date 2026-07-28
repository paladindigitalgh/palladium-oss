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

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authhttpapi "github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	authpostgres "github.com/paladindigitalgh/palladium-oss/internal/auth/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	"github.com/paladindigitalgh/palladium-oss/internal/config"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	customerservice "github.com/paladindigitalgh/palladium-oss/internal/customer/service"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/health"
	"github.com/paladindigitalgh/palladium-oss/internal/httpserver"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/service"
	locationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
	locationpostgres "github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	locationservice "github.com/paladindigitalgh/palladium-oss/internal/location/service"
	logging "github.com/paladindigitalgh/palladium-oss/internal/log"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
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

	// Site is the only Inventory entity with an HTTP surface so far (see
	// this milestone's scope); Building, Room, Rack, and Device follow the
	// same repository -> service -> handler chain once their own
	// endpoints exist. clock.New() and id.New() are shared across
	// repositories deliberately: they are stateless, so there is no
	// reason for each repository to hold its own instance.
	siteRepo := inventorypostgres.NewSiteRepository(pool, clock.New(), id.New())
	siteService := service.NewSiteService(siteRepo)
	siteHandler := httpapi.NewSiteHandler(siteService)

	// Customer follows the exact same repository -> service -> handler
	// chain as Site, one domain package over (internal/customer instead
	// of internal/inventory).
	customerRepo := customerpostgres.NewCustomerRepository(pool, clock.New(), id.New())
	customerSvc := customerservice.NewCustomerService(customerRepo)
	customerHandler := customerhttpapi.NewCustomerHandler(customerSvc)

	// Location follows the exact same repository -> service -> handler
	// chain as Site and Customer, one domain package over
	// (internal/location instead of internal/customer).
	locationRepo := locationpostgres.NewLocationRepository(pool, clock.New(), id.New())
	locationSvc := locationservice.NewLocationService(locationRepo)
	locationHandler := locationhttpapi.NewLocationHandler(locationSvc)

	// tokenIssuer is shared by auth.Middleware (validates incoming tokens)
	// and LoginHandler (issues new ones): both need to agree on the same
	// secret and expiration, and a single instance is the simplest way to
	// guarantee that rather than constructing it twice from the same
	// config values.
	tokenIssuer := auth.NewTokenIssuer([]byte(cfg.JWT.Secret), cfg.JWT.Expiration, clock.New())

	userRepo := authpostgres.NewUserRepository(pool, clock.New(), id.New())
	authService := auth.NewAuthService(userRepo, tokenIssuer)
	loginHandler := authhttpapi.NewLoginHandler(authService, cfg.JWT.Expiration)

	// authz.Middleware reuses userRepo (the same UserRepository
	// authService already depends on) rather than a second instance —
	// there is no reason for two, and sharing makes it obvious both are
	// always looking at the same table.
	authzMiddleware := authz.NewMiddleware(userRepo)

	router := api.NewRouter(api.Dependencies{
		Logger:          logger,
		HealthCheckers:  healthCheckers,
		Version:         version.Version,
		Commit:          version.Commit,
		SiteHandler:     siteHandler,
		CustomerHandler: customerHandler,
		LocationHandler: locationHandler,
		Tokens:          tokenIssuer,
		LoginHandler:    loginHandler,
		Authz:           authzMiddleware,
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
