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
	cataloghttpapi "github.com/paladindigitalgh/palladium-oss/internal/catalog/httpapi"
	catalogpostgres "github.com/paladindigitalgh/palladium-oss/internal/catalog/postgres"
	catalogservice "github.com/paladindigitalgh/palladium-oss/internal/catalog/service"
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
	producthttpapi "github.com/paladindigitalgh/palladium-oss/internal/product/httpapi"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	productservice "github.com/paladindigitalgh/palladium-oss/internal/product/service"
	provisioninghttpapi "github.com/paladindigitalgh/palladium-oss/internal/provisioning/httpapi"
	provisioningpostgres "github.com/paladindigitalgh/palladium-oss/internal/provisioning/postgres"
	provisioningservice "github.com/paladindigitalgh/palladium-oss/internal/provisioning/service"
	api "github.com/paladindigitalgh/palladium-oss/internal/server"
	servicehttpapi "github.com/paladindigitalgh/palladium-oss/internal/service/httpapi"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
	serviceservice "github.com/paladindigitalgh/palladium-oss/internal/service/service"
	serviceequipmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/httpapi"
	serviceequipmentpostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/postgres"
	serviceequipmentservice "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/service"
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

	// Catalog and Product follow the exact same repository -> service ->
	// handler chain as every domain above, two packages over
	// (internal/catalog and internal/product instead of
	// internal/location). Product is constructed after Catalog, mirroring
	// the FK dependency between their tables, though nothing here actually
	// requires that ordering — each repository only needs the shared pool.
	catalogRepo := catalogpostgres.NewCatalogRepository(pool, clock.New(), id.New())
	catalogSvc := catalogservice.NewCatalogService(catalogRepo)
	catalogHandler := cataloghttpapi.NewCatalogHandler(catalogSvc)

	productRepo := productpostgres.NewProductRepository(pool, clock.New(), id.New())
	productSvc := productservice.NewProductService(productRepo)
	productHandler := producthttpapi.NewProductHandler(productSvc)

	// Service follows the exact same repository -> service -> handler
	// chain as every domain above, one package over (internal/service
	// instead of internal/product). It is constructed after Location,
	// Catalog, and Product, mirroring the two foreign keys a Service row
	// requires, though as with Product/Catalog above nothing here
	// actually requires that ordering.
	serviceRepo := servicepostgres.NewServiceRepository(pool, clock.New(), id.New())
	serviceSvc := serviceservice.NewServiceService(serviceRepo)
	serviceHandler := servicehttpapi.NewServiceHandler(serviceSvc)

	// Service Equipment follows the exact same repository -> service ->
	// handler chain as every domain above, one package over
	// (internal/serviceequipment instead of internal/service). Its two
	// foreign keys are Service and inventory.Device; Device has no
	// repository constructed above (it has no HTTP surface — see this
	// milestone's scope), so nothing here needs a prior devicepostgres
	// variable the way Service needed locationRepo/productRepo.
	serviceEquipmentRepo := serviceequipmentpostgres.NewServiceEquipmentRepository(pool, clock.New(), id.New())
	serviceEquipmentSvc := serviceequipmentservice.NewServiceEquipmentService(serviceEquipmentRepo)
	serviceEquipmentHandler := serviceequipmenthttpapi.NewServiceEquipmentHandler(serviceEquipmentSvc)

	// Provisioning follows the same repository -> service -> handler
	// chain as every domain above, but ProvisioningService additionally
	// takes a clock.Clock — the one business logic layer in this codebase
	// that needs one, since it stamps StartedAt/CompletedAt itself as
	// part of enforcing state transitions (see
	// internal/provisioning/service's package doc comment). A separate
	// clock.New() is passed here rather than reusing one of the instances
	// above only because each repository already gets its own by
	// convention (see the comment on siteRepo above); there is no
	// shared-state reason it has to be this specific instance.
	provisioningRepo := provisioningpostgres.NewProvisioningRepository(pool, clock.New(), id.New())
	provisioningSvc := provisioningservice.NewProvisioningService(provisioningRepo, clock.New())
	provisioningHandler := provisioninghttpapi.NewProvisioningHandler(provisioningSvc)

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
		Logger:                  logger,
		HealthCheckers:          healthCheckers,
		Version:                 version.Version,
		Commit:                  version.Commit,
		SiteHandler:             siteHandler,
		CustomerHandler:         customerHandler,
		LocationHandler:         locationHandler,
		CatalogHandler:          catalogHandler,
		ProductHandler:          productHandler,
		ServiceHandler:          serviceHandler,
		ServiceEquipmentHandler: serviceEquipmentHandler,
		ProvisioningHandler:     provisioningHandler,
		Tokens:                  tokenIssuer,
		LoginHandler:            loginHandler,
		Authz:                   authzMiddleware,
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
