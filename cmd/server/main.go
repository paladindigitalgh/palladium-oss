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

	accessattachmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessattachment/httpapi"
	accessattachmentpostgres "github.com/paladindigitalgh/palladium-oss/internal/accessattachment/postgres"
	accessattachmentservice "github.com/paladindigitalgh/palladium-oss/internal/accessattachment/service"
	accessinterfacehttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessinterface/httpapi"
	accessinterfacepostgres "github.com/paladindigitalgh/palladium-oss/internal/accessinterface/postgres"
	accessinterfaceservice "github.com/paladindigitalgh/palladium-oss/internal/accessinterface/service"
	accessnetworkhttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/httpapi"
	accessnetworkpostgres "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/postgres"
	accessnetworkservice "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/service"
	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authhttpapi "github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	authpostgres "github.com/paladindigitalgh/palladium-oss/internal/auth/postgres"
	authenticationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/authentication/httpapi"
	authenticationpostgres "github.com/paladindigitalgh/palladium-oss/internal/authentication/postgres"
	authenticationservice "github.com/paladindigitalgh/palladium-oss/internal/authentication/service"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	cataloghttpapi "github.com/paladindigitalgh/palladium-oss/internal/catalog/httpapi"
	catalogpostgres "github.com/paladindigitalgh/palladium-oss/internal/catalog/postgres"
	catalogservice "github.com/paladindigitalgh/palladium-oss/internal/catalog/service"
	"github.com/paladindigitalgh/palladium-oss/internal/config"
	connectionprofilehttpapi "github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/httpapi"
	connectionprofilepostgres "github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/postgres"
	connectionprofileservice "github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/service"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	customerservice "github.com/paladindigitalgh/palladium-oss/internal/customer/service"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
	diagnosticshttpapi "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/httpapi"
	diagnosticsservice "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/service"
	eventhttpapi "github.com/paladindigitalgh/palladium-oss/internal/event/httpapi"
	eventpostgres "github.com/paladindigitalgh/palladium-oss/internal/event/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/health"
	"github.com/paladindigitalgh/palladium-oss/internal/httpserver"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/service"
	locationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
	locationpostgres "github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	locationservice "github.com/paladindigitalgh/palladium-oss/internal/location/service"
	logging "github.com/paladindigitalgh/palladium-oss/internal/log"
	olthttpapi "github.com/paladindigitalgh/palladium-oss/internal/olt/httpapi"
	oltpostgres "github.com/paladindigitalgh/palladium-oss/internal/olt/postgres"
	oltservice "github.com/paladindigitalgh/palladium-oss/internal/olt/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/encryption"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/retry"
	"github.com/paladindigitalgh/palladium-oss/internal/plugin"
	pluginmock "github.com/paladindigitalgh/palladium-oss/internal/plugin/mock"
	ponporthttpapi "github.com/paladindigitalgh/palladium-oss/internal/ponport/httpapi"
	ponportpostgres "github.com/paladindigitalgh/palladium-oss/internal/ponport/postgres"
	ponportservice "github.com/paladindigitalgh/palladium-oss/internal/ponport/service"
	producthttpapi "github.com/paladindigitalgh/palladium-oss/internal/product/httpapi"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	productservice "github.com/paladindigitalgh/palladium-oss/internal/product/service"
	api "github.com/paladindigitalgh/palladium-oss/internal/server"
	servicehttpapi "github.com/paladindigitalgh/palladium-oss/internal/service/httpapi"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
	serviceservice "github.com/paladindigitalgh/palladium-oss/internal/service/service"
	serviceequipmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/httpapi"
	serviceequipmentpostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/postgres"
	serviceequipmentservice "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/service"
	serviceprofilehttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/httpapi"
	serviceprofilepostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/postgres"
	serviceprofileservice "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/service"
	"github.com/paladindigitalgh/palladium-oss/internal/version"
	workflowengine "github.com/paladindigitalgh/palladium-oss/internal/workflow/engine"
	workflowhttpapi "github.com/paladindigitalgh/palladium-oss/internal/workflow/httpapi"
	workflowpostgres "github.com/paladindigitalgh/palladium-oss/internal/workflow/postgres"
	workflowservice "github.com/paladindigitalgh/palladium-oss/internal/workflow/service"
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

	// Site and Device are the only Inventory entities with an HTTP surface
	// so far; Building and Room follow the same repository -> service ->
	// handler chain once their own endpoints exist. clock.New() and
	// id.New() are shared across repositories deliberately: they are
	// stateless, so there is no reason for each repository to hold its
	// own instance.
	siteRepo := inventorypostgres.NewSiteRepository(pool, clock.New(), id.New())
	siteService := service.NewSiteService(siteRepo)
	siteHandler := httpapi.NewSiteHandler(siteService)

	// Device follows the exact same repository -> service -> handler
	// chain as Site, one entity over in the same Inventory hierarchy.
	deviceRepo := inventorypostgres.NewDeviceRepository(pool, clock.New(), id.New())
	deviceService := service.NewDeviceService(deviceRepo)
	deviceHandler := httpapi.NewDeviceHandler(deviceService)

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

	// Service Profile follows the exact same repository -> service ->
	// handler chain as every domain above, mirroring internal/catalog's
	// own standalone (non-nested) shape.
	serviceProfileRepo := serviceprofilepostgres.NewServiceProfileRepository(pool, clock.New(), id.New())
	serviceProfileSvc := serviceprofileservice.NewServiceProfileService(serviceProfileRepo)
	serviceProfileHandler := serviceprofilehttpapi.NewServiceProfileHandler(serviceProfileSvc)

	// Service follows the exact same repository -> service -> handler
	// chain as every domain above, one package over (internal/service
	// instead of internal/product). It is constructed after Location,
	// Catalog, Product, and Service Profile, mirroring the three foreign
	// keys a Service row requires, though as with Product/Catalog above
	// nothing here actually requires that ordering.
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

	// Event has no service layer: there is no business logic beyond
	// append and list (see internal/event's package doc comment), so the
	// repository is wired directly to the handler.
	eventRepo := eventpostgres.NewEventRepository(pool, clock.New(), id.New())
	eventHandler := eventhttpapi.NewEventHandler(eventRepo)

	// pluginRegistry is built and populated with every available plugin
	// once, at startup — the same "every Register call happens before
	// the HTTP server starts" assumption
	// internal/plugin.DefaultRegistry's own doc comment documents.
	// MockPlugin is the only plugin registered today: there is no real
	// OLT/router vendor plugin yet, so every workflow capability is
	// fulfilled by the simulated vendor until one is built.
	pluginRegistry := plugin.NewDefaultRegistry()
	pluginRegistry.Register(pluginmock.NewMockPlugin(logger))

	// Workflow follows the same repository -> service -> handler chain
	// as every domain above, but its service layer additionally takes
	// clock.Clock (it stamps StartedAt/CompletedAt as part of enforcing
	// state transitions) and an event.EventRepository (every transition
	// records an Event — see internal/workflow/service's package doc
	// comment). Engine sits alongside the service layer, depending on it
	// directly (not the repository) so that executing a workflow reuses
	// the exact same transition-and-event-recording logic as any other
	// caller (see internal/workflow/engine's package doc comment).
	workflowRepo := workflowpostgres.NewRepository(pool, clock.New(), id.New())
	workflowSvc := workflowservice.New(workflowRepo, eventRepo, clock.New())
	workflowEngine := workflowengine.NewDefaultEngine(workflowSvc, serviceRepo, serviceEquipmentRepo, pluginRegistry, clock.New())
	workflowHandler := workflowhttpapi.NewWorkflowHandler(workflowSvc, workflowEngine)

	// Access Network, OLT, and PON Port follow the same repository ->
	// service -> handler chain as every domain above, three packages
	// over (internal/accessnetwork, internal/olt, and internal/ponport).
	// OLT is constructed after AccessNetwork and PONPort after OLT,
	// mirroring the FK chain between their tables, though as with every
	// other pair above nothing here actually requires that ordering.
	accessNetworkRepo := accessnetworkpostgres.NewAccessNetworkRepository(pool, clock.New(), id.New())
	accessNetworkSvc := accessnetworkservice.NewAccessNetworkService(accessNetworkRepo)
	accessNetworkHandler := accessnetworkhttpapi.NewAccessNetworkHandler(accessNetworkSvc)

	oltRepo := oltpostgres.NewOLTRepository(pool, clock.New(), id.New())
	oltSvc := oltservice.NewOLTService(oltRepo)
	oltHandler := olthttpapi.NewOLTHandler(oltSvc)

	ponPortRepo := ponportpostgres.NewPONPortRepository(pool, clock.New(), id.New())
	ponPortSvc := ponportservice.NewPONPortService(ponPortRepo)
	ponPortHandler := ponporthttpapi.NewPONPortHandler(ponPortSvc)

	// Access Interface and Access Attachment follow the same repository
	// -> service -> handler chain as every domain above, two packages
	// over (internal/accessinterface and internal/accessattachment).
	// AccessAttachment is constructed after AccessInterface, mirroring
	// the FK from access_attachments into access_interfaces, though as
	// with every other pair above nothing here actually requires that
	// ordering.
	accessInterfaceRepo := accessinterfacepostgres.NewAccessInterfaceRepository(pool, clock.New(), id.New())
	accessInterfaceSvc := accessinterfaceservice.NewAccessInterfaceService(accessInterfaceRepo)
	accessInterfaceHandler := accessinterfacehttpapi.NewAccessInterfaceHandler(accessInterfaceSvc)

	accessAttachmentRepo := accessattachmentpostgres.NewAccessAttachmentRepository(pool, clock.New(), id.New())
	accessAttachmentSvc := accessattachmentservice.NewAccessAttachmentService(accessAttachmentRepo)
	accessAttachmentHandler := accessattachmenthttpapi.NewAccessAttachmentHandler(accessAttachmentSvc)

	// Diagnostics has no repository at all — see
	// internal/diagnostics/service's doc comment on why: this milestone's
	// framework performs no persistence, so there is nothing for a
	// postgres package to do. The registry is built and populated with
	// this milestone's one built-in diagnostic right here, at startup,
	// the same "every Register call happens once, before the HTTP server
	// starts" assumption internal/diagnostics.DefaultRegistry's own doc
	// comment documents.
	diagnosticsRegistry := diagnostics.NewDefaultRegistry()
	diagnosticsRegistry.Register(diagnostics.NewBasicONUCheck())
	diagnosticsSvc := diagnosticsservice.NewDiagnosticsService(diagnosticsRegistry)
	diagnosticsHandler := diagnosticshttpapi.NewDiagnosticsHandler(diagnosticsSvc)

	// encryptor is shared by every repository that stores an encrypted
	// secret — only internal/authentication/postgres today, but any
	// future infrastructure package that needs its own encrypted field
	// (rather than referencing an Authentication record by ID) would take
	// the same instance rather than parsing PALLADIUM_MASTER_KEY again.
	// cfg.Validate (see internal/config/config.go) already guarantees
	// MasterKey is non-empty and, in production, not the insecure
	// checked-in default, so the only remaining failure mode here is a
	// malformed base64 value — a startup-time configuration error, not a
	// runtime condition to recover from.
	encryptor, err := encryption.NewAESGCMEncryptorFromBase64Key(cfg.Encryption.MasterKey)
	if err != nil {
		return fmt.Errorf("build encryptor: %w", err)
	}

	// Authentication follows the same repository -> service -> handler
	// chain as every domain above, but the repository additionally takes
	// encryptor: it encrypts Password/PrivateKey before every write and
	// decrypts them after every read, so the plaintext never crosses the
	// service or HTTP layers on the way in or out of PostgreSQL (see
	// internal/authentication/postgres's package doc comment).
	authenticationRepo := authenticationpostgres.NewAuthenticationRepository(pool, clock.New(), id.New(), encryptor)
	authenticationSvc := authenticationservice.NewAuthenticationService(authenticationRepo)
	authenticationHandler := authenticationhttpapi.NewAuthenticationHandler(authenticationSvc)

	// Connection Profile follows the same repository -> service -> handler
	// chain as every domain above, constructed after Authentication,
	// mirroring the FK from connection_profiles into
	// authentication_methods, though as with every other pair in this
	// file nothing here actually requires that ordering.
	connectionProfileRepo := connectionprofilepostgres.NewConnectionProfileRepository(pool, clock.New(), id.New())
	connectionProfileSvc := connectionprofileservice.NewConnectionProfileService(connectionProfileRepo)
	connectionProfileHandler := connectionprofilehttpapi.NewConnectionProfileHandler(connectionProfileSvc)

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
		Logger:                   logger,
		HealthCheckers:           healthCheckers,
		Version:                  version.Version,
		Commit:                   version.Commit,
		SiteHandler:              siteHandler,
		DeviceHandler:            deviceHandler,
		CustomerHandler:          customerHandler,
		LocationHandler:          locationHandler,
		CatalogHandler:           catalogHandler,
		ProductHandler:           productHandler,
		ServiceProfileHandler:    serviceProfileHandler,
		DiagnosticsHandler:       diagnosticsHandler,
		ServiceHandler:           serviceHandler,
		ServiceEquipmentHandler:  serviceEquipmentHandler,
		WorkflowHandler:          workflowHandler,
		EventHandler:             eventHandler,
		AccessNetworkHandler:     accessNetworkHandler,
		OLTHandler:               oltHandler,
		PONPortHandler:           ponPortHandler,
		AccessInterfaceHandler:   accessInterfaceHandler,
		AccessAttachmentHandler:  accessAttachmentHandler,
		AuthenticationHandler:    authenticationHandler,
		ConnectionProfileHandler: connectionProfileHandler,
		Tokens:                   tokenIssuer,
		LoginHandler:             loginHandler,
		Authz:                    authzMiddleware,
		AllowedOrigin:            cfg.HTTP.AllowedOrigin,
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
