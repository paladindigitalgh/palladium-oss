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
	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	accesstopologyhttpapi "github.com/paladindigitalgh/palladium-oss/internal/accesstopology/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authhttpapi "github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	authpostgres "github.com/paladindigitalgh/palladium-oss/internal/auth/postgres"
	authservice "github.com/paladindigitalgh/palladium-oss/internal/auth/service"
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
	contacthttpapi "github.com/paladindigitalgh/palladium-oss/internal/contact/httpapi"
	contactpostgres "github.com/paladindigitalgh/palladium-oss/internal/contact/postgres"
	contactservice "github.com/paladindigitalgh/palladium-oss/internal/contact/service"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	customerremoval "github.com/paladindigitalgh/palladium-oss/internal/customer/removal"
	customerservice "github.com/paladindigitalgh/palladium-oss/internal/customer/service"
	customerdevicehttpapi "github.com/paladindigitalgh/palladium-oss/internal/customerdevice/httpapi"
	customerdevicepostgres "github.com/paladindigitalgh/palladium-oss/internal/customerdevice/postgres"
	customerdeviceservice "github.com/paladindigitalgh/palladium-oss/internal/customerdevice/service"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	devicemanufacturerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer/httpapi"
	devicemanufacturerpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer/postgres"
	devicemanufacturerservice "github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer/service"
	devicemodelhttpapi "github.com/paladindigitalgh/palladium-oss/internal/devicemodel/httpapi"
	devicemodelpostgres "github.com/paladindigitalgh/palladium-oss/internal/devicemodel/postgres"
	devicemodelservice "github.com/paladindigitalgh/palladium-oss/internal/devicemodel/service"
	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
	diagnosticshttpapi "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/httpapi"
	kontronhttpapi "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron/httpapi"
	kontronservice "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron/service"
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
	notehttpapi "github.com/paladindigitalgh/palladium-oss/internal/note/httpapi"
	notepostgres "github.com/paladindigitalgh/palladium-oss/internal/note/postgres"
	noteservice "github.com/paladindigitalgh/palladium-oss/internal/note/service"
	"github.com/paladindigitalgh/palladium-oss/internal/olt/connect"
	olthttpapi "github.com/paladindigitalgh/palladium-oss/internal/olt/httpapi"
	oltpostgres "github.com/paladindigitalgh/palladium-oss/internal/olt/postgres"
	oltservice "github.com/paladindigitalgh/palladium-oss/internal/olt/service"
	oltmodelhttpapi "github.com/paladindigitalgh/palladium-oss/internal/oltmodel/httpapi"
	oltmodelpostgres "github.com/paladindigitalgh/palladium-oss/internal/oltmodel/postgres"
	oltmodelservice "github.com/paladindigitalgh/palladium-oss/internal/oltmodel/service"
	onuauthorizationpostgres "github.com/paladindigitalgh/palladium-oss/internal/onuauthorization/postgres"
	onuauthorizationservice "github.com/paladindigitalgh/palladium-oss/internal/onuauthorization/service"
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
	providerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/provider/httpapi"
	providerpostgres "github.com/paladindigitalgh/palladium-oss/internal/provider/postgres"
	providerservice "github.com/paladindigitalgh/palladium-oss/internal/provider/service"
	provisioninghttpapi "github.com/paladindigitalgh/palladium-oss/internal/provisioning/httpapi"
	provisioningkontronhttpapi "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/httpapi"
	provisioningkontronplugin "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/plugin"
	provisioningkontronservice "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron/service"
	provisioningpostgres "github.com/paladindigitalgh/palladium-oss/internal/provisioning/postgres"
	provisioningservice "github.com/paladindigitalgh/palladium-oss/internal/provisioning/service"
	reporthttpapi "github.com/paladindigitalgh/palladium-oss/internal/report/httpapi"
	reportpostgres "github.com/paladindigitalgh/palladium-oss/internal/report/postgres"
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
	workflowworker "github.com/paladindigitalgh/palladium-oss/internal/workflow/worker"
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

	// Building and Room follow the exact same repository -> service ->
	// handler chain as Site, one and two entities over in the same
	// Inventory hierarchy.
	buildingRepo := inventorypostgres.NewBuildingRepository(pool, clock.New(), id.New())
	buildingService := service.NewBuildingService(buildingRepo)
	buildingHandler := httpapi.NewBuildingHandler(buildingService)

	roomRepo := inventorypostgres.NewRoomRepository(pool, clock.New(), id.New())
	roomService := service.NewRoomService(roomRepo)
	roomHandler := httpapi.NewRoomHandler(roomService)

	rackRepo := inventorypostgres.NewRackRepository(pool, clock.New(), id.New())
	rackService := service.NewRackService(rackRepo)
	rackHandler := httpapi.NewRackHandler(rackService)

	// Device follows the exact same repository -> service -> handler
	// chain as Site, one entity over in the same Inventory hierarchy.
	deviceRepo := inventorypostgres.NewDeviceRepository(pool, clock.New(), id.New())
	deviceService := service.NewDeviceService(deviceRepo)
	deviceHandler := httpapi.NewDeviceHandler(deviceService)

	// DeviceManufacturer and DeviceModel are constructed before Device's
	// handler is used, mirroring OLTModel/PONPort's own construction
	// order relative to OLT below: Device.DeviceModelID references
	// DeviceModel, which references DeviceManufacturer, so both catalogs
	// exist before anything that looks them up runs.
	deviceManufacturerRepo := devicemanufacturerpostgres.NewDeviceManufacturerRepository(pool, clock.New(), id.New())
	deviceManufacturerSvc := devicemanufacturerservice.NewDeviceManufacturerService(deviceManufacturerRepo)
	deviceManufacturerHandler := devicemanufacturerhttpapi.NewDeviceManufacturerHandler(deviceManufacturerSvc)

	deviceModelRepo := devicemodelpostgres.NewDeviceModelRepository(pool, clock.New(), id.New())
	deviceModelSvc := devicemodelservice.NewDeviceModelService(deviceModelRepo)
	deviceModelHandler := devicemodelhttpapi.NewDeviceModelHandler(deviceModelSvc)

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

	// Contact follows the exact same repository -> service -> handler
	// chain as Location, one domain package over (internal/contact
	// instead of internal/location).
	contactRepo := contactpostgres.NewContactRepository(pool, clock.New(), id.New())
	contactSvc := contactservice.NewContactService(contactRepo)
	contactHandler := contacthttpapi.NewContactHandler(contactSvc)

	// Catalog and Product follow the exact same repository -> service ->
	// handler chain as every domain above, two packages over
	// (internal/catalog and internal/product instead of
	// internal/location). Product is constructed after Catalog, mirroring
	// the FK dependency between their tables, though nothing here actually
	// requires that ordering — each repository only needs the shared pool.
	catalogRepo := catalogpostgres.NewCatalogRepository(pool, clock.New(), id.New())
	catalogSvc := catalogservice.NewCatalogService(catalogRepo)
	catalogHandler := cataloghttpapi.NewCatalogHandler(catalogSvc)

	// Provider follows the exact same repository -> service -> handler
	// chain as every domain above, mirroring internal/serviceprofile's
	// own standalone (non-nested) shape -- constructed before Product
	// since products.provider_id references providers(id), though
	// nothing here actually requires that ordering.
	providerRepo := providerpostgres.NewProviderRepository(pool, clock.New(), id.New())
	providerSvc := providerservice.NewProviderService(providerRepo)
	providerHandler := providerhttpapi.NewProviderHandler(providerSvc)

	productRepo := productpostgres.NewProductRepository(pool, clock.New(), id.New())
	productSvc := productservice.NewProductService(productRepo)
	productHandler := producthttpapi.NewProductHandler(productSvc)

	// Provisioning Profile follows the exact same repository -> service ->
	// handler chain as Product, one package over (internal/provisioning
	// instead of internal/product) -- see that package's own doc comment
	// for why this is not the internal/provisioning that existed earlier
	// in this codebase's history.
	provisioningProfileRepo := provisioningpostgres.NewProvisioningProfileRepository(pool, clock.New(), id.New())
	provisioningProfileSvc := provisioningservice.NewProvisioningProfileService(provisioningProfileRepo)
	provisioningProfileHandler := provisioninghttpapi.NewProvisioningProfileHandler(provisioningProfileSvc)

	// Service Profile follows the exact same repository -> service ->
	// handler chain as every domain above, mirroring internal/catalog's
	// own standalone (non-nested) shape.
	serviceProfileRepo := serviceprofilepostgres.NewServiceProfileRepository(pool, clock.New(), id.New())
	serviceProfileSvc := serviceprofileservice.NewServiceProfileService(serviceProfileRepo)
	serviceProfileHandler := serviceprofilehttpapi.NewServiceProfileHandler(serviceProfileSvc)

	// Service Equipment's repository is built here, ahead of Service
	// itself, even though the "repository -> service -> handler" chain
	// below would normally build it alongside Service Equipment's own
	// service (see the comment near serviceEquipmentSvc's construction
	// further down): ServiceService.Delete now depends on it too, to give
	// a specific "still has equipment attached" error instead of a
	// generic foreign-key one (see that method's own doc comment) — one
	// more reader of the repository nothing here actually requires any
	// particular construction order for.
	serviceEquipmentRepo := serviceequipmentpostgres.NewServiceEquipmentRepository(pool, clock.New(), id.New())

	// Service follows the exact same repository -> service -> handler
	// chain as every domain above, one package over (internal/service
	// instead of internal/product). It is constructed after Location,
	// Catalog, Product, and Service Profile, mirroring the three foreign
	// keys a Service row requires, though as with Product/Catalog above
	// nothing here actually requires that ordering.
	serviceRepo := servicepostgres.NewServiceRepository(pool, clock.New(), id.New())
	serviceSvc := serviceservice.NewServiceService(serviceRepo, serviceEquipmentRepo)
	serviceHandler := servicehttpapi.NewServiceHandler(serviceSvc)

	// Customer Device needs Service Equipment's repository too (not
	// service — see below), for the same "read into the other's domain"
	// reasoning CustomerDeviceService.Update gives: it must check "does
	// this Device still fulfill an active Service" before allowing a
	// detach, and ServiceEquipmentService's own markDeviceUnused
	// (constructed further below) makes the same check in reverse — "is
	// this Device still attached to a Customer" — against
	// customerDeviceRepo, built here alongside it for the identical
	// reason.
	customerDeviceRepo := customerdevicepostgres.NewCustomerDeviceRepository(pool, clock.New(), id.New())

	// Service Equipment's own service and handler are built much further
	// below (see the comment beside serviceEquipmentSvc's construction),
	// not here alongside its repository: ServiceEquipmentService now also
	// depends on onuAuthorizationSvc, accessInterfaceSvc, and
	// accessAttachmentSvc, none of which exist yet at this point in
	// construction. serviceEquipmentRepo itself is still built here,
	// though, since workflowEngine, customerDeviceSvc, and
	// customerResolver below all need the repository (not the service)
	// well before that point.

	// Customer Device follows the exact same repository -> service ->
	// handler chain as Service Equipment, one domain up
	// (internal/customerdevice instead of internal/serviceequipment): its
	// foreign keys are Customer, inventory.Device, and (optionally)
	// Location, the Device one reusing deviceService for the same "run
	// the real business-rule check, not the raw repository" reason
	// serviceEquipmentSvc does. locationSvc backs the same check for
	// LocationID -- a set LocationID must belong to the same Customer.
	customerDeviceSvc := customerdeviceservice.NewCustomerDeviceService(
		customerDeviceRepo, deviceService, deviceService, serviceEquipmentRepo, locationSvc)
	customerDeviceHandler := customerdevicehttpapi.NewCustomerDeviceHandler(customerDeviceSvc)

	// Event has no service layer: there is no business logic beyond
	// append and list (see internal/event's package doc comment), so the
	// repository is wired directly to the handler.
	eventRepo := eventpostgres.NewEventRepository(pool, clock.New(), id.New())
	eventHandler := eventhttpapi.NewEventHandler(eventRepo)

	// userRepo is built here, well before the rest of auth's wiring
	// further down this file, so NoteHandler (immediately below) can use
	// it to resolve an author's current FirstName/LastName at Create time
	// (see note_handler.go's own doc comment on why that snapshot can't
	// come from JWT claims). The later auth wiring reuses this exact
	// instance rather than constructing a second one — see its own
	// comment for why sharing is preferred.
	userRepo := authpostgres.NewUserRepository(pool, clock.New(), id.New())

	// Note, unlike Event, gets a real service layer: Create validates a
	// client-submitted body (see internal/note/service), the same
	// reasoning every other client-writable domain in this file gets one
	// and Event does not.
	noteRepo := notepostgres.NewNoteRepository(pool, clock.New(), id.New())
	noteSvc := noteservice.NewNoteService(noteRepo)
	noteHandler := notehttpapi.NewNoteHandler(noteSvc, userRepo)

	// Report, like Event, has no service layer: every method is a read
	// query with no business logic (see internal/report's own package
	// doc comment), and needs neither clock nor id.Generator since it
	// never writes anything.
	reportRepo := reportpostgres.NewRepository(pool)
	reportHandler := reporthttpapi.NewReportHandler(reportRepo)

	// pluginRegistry is built and populated with every available plugin
	// once, at startup — the same "every Register call happens before
	// the HTTP server starts" assumption
	// internal/plugin.DefaultRegistry's own doc comment documents.
	// MockPlugin is registered first, for every capability, so every
	// workflow capability defaults to the simulated vendor. The real
	// Kontron plugin (registered further below, once its dependencies —
	// oltRepo, oltModelRepo, kontronDialer, accessTopologyResolver,
	// provisioningProfileRepo — are built) is registered afterward and,
	// per Registry.Register's documented last-write-wins-per-capability
	// semantics, replaces MockPlugin for plugin.ProvisionService,
	// plugin.ResumeService, plugin.SuspendService, and
	// plugin.DisconnectService — the four capabilities it declares.
	// Reprovision/Synchronize still run through MockPlugin until real
	// Kontron support for those exists.
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
	workflowHandler := workflowhttpapi.NewWorkflowHandler(workflowSvc)

	// The worker is the Workflow Engine's job queue (TASKS.md Phase 7):
	// the one process that ever calls workflowEngine.Execute now (see
	// internal/workflow/httpapi's package doc comment for why the HTTP
	// layer no longer does). It polls workflowRepo directly, not
	// workflowSvc, since NextPending is a plain query with no
	// transition/event-recording semantics for workflowSvc to add.
	workflowWorker := workflowworker.New(workflowRepo, workflowEngine, cfg.Workflow.PollInterval, logger)

	// Access Network, OLT, and PON Port follow the same repository ->
	// service -> handler chain as every domain above, three packages
	// over (internal/accessnetwork, internal/olt, and internal/ponport).
	// OLT is constructed after AccessNetwork and PONPort after OLT,
	// mirroring the FK chain between their tables, though as with every
	// other pair above nothing here actually requires that ordering.
	accessNetworkRepo := accessnetworkpostgres.NewAccessNetworkRepository(pool, clock.New(), id.New())
	accessNetworkSvc := accessnetworkservice.NewAccessNetworkService(accessNetworkRepo)
	accessNetworkHandler := accessnetworkhttpapi.NewAccessNetworkHandler(accessNetworkSvc)

	// OLTModel and PONPort are both constructed before OLT, not after
	// (unlike AccessNetwork/OLT/PONPort's usual FK-mirroring order just
	// above): OLTService.Create depends on oltModelRepo and ponPortRepo
	// directly to auto-create PON ports on OLT creation (see
	// internal/olt/service.OLTService's own doc comment), so both must
	// already exist by the time NewOLTService is called.
	oltModelRepo := oltmodelpostgres.NewOLTModelRepository(pool, clock.New(), id.New())
	oltModelSvc := oltmodelservice.NewOLTModelService(oltModelRepo)
	oltModelHandler := oltmodelhttpapi.NewOLTModelHandler(oltModelSvc)

	ponPortRepo := ponportpostgres.NewPONPortRepository(pool, clock.New(), id.New())
	ponPortSvc := ponportservice.NewPONPortService(ponPortRepo)
	ponPortHandler := ponporthttpapi.NewPONPortHandler(ponPortSvc)

	oltRepo := oltpostgres.NewOLTRepository(pool, clock.New(), id.New())
	oltSvc := oltservice.NewOLTService(oltRepo, oltModelRepo, ponPortRepo)
	oltHandler := olthttpapi.NewOLTHandler(oltSvc)

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

	// Kontron/Iskratel C16 diagnostics (internal/diagnostics/kontron)
	// reuse oltRepo, connectionProfileRepo, and authenticationRepo
	// directly rather than a repository of their own: resolving how to
	// reach a specific OLT is connect.Dialer's whole job (see that
	// package's own doc comment), and internal/diagnostics/kontron/service
	// has nothing of its own to persist (see that package's own doc
	// comment). This is deliberately not wired into diagnosticsRegistry
	// above — see internal/diagnostics/kontron's package doc comment on
	// why it is a separate, purpose-built framework rather than an
	// extension of internal/diagnostics's Request{ONUID}-shaped one.
	// oltRepo and oltModelRepo are reused again here (already built above
	// for oltSvc): AggregatedBlacklist needs the same "which OLTs exist,
	// which of those are Kontron" answer OLTService.Create's own
	// auto-port-creation cascade already needs from them.
	kontronDialer := connect.NewDialer(oltRepo, connectionProfileRepo, authenticationRepo, cfg.SSH.KnownHostsFile)
	kontronSvc := kontronservice.NewKontronService(kontronDialer, oltRepo, oltModelRepo)
	kontronHandler := kontronhttpapi.NewKontronHandler(kontronSvc)

	// ONU Authorization (internal/onuauthorization) records that a Device
	// has been authorized directly on an OLT, independent of any Service
	// assignment — see that package's own doc comment for the gap this
	// closes. Built here, alongside deviceRepo/deviceService above, since
	// this milestone's Kontron write-command services below need it
	// (AuthorizeAndCreateDeviceService to persist the record,
	// DeauthorizationService to resolve it when a Device has no
	// ServiceEquipment), and so does serviceEquipmentSvc just below, to
	// look an existing record up again.
	onuAuthorizationRepo := onuauthorizationpostgres.NewOnuAuthorizationRepository(pool, clock.New(), id.New())
	onuAuthorizationSvc := onuauthorizationservice.NewOnuAuthorizationService(onuAuthorizationRepo)

	// Service Equipment's service and handler are built here, not
	// alongside serviceEquipmentRepo above, because
	// ServiceEquipmentService now depends on accessInterfaceSvc,
	// accessAttachmentSvc, and onuAuthorizationSvc (all built above) for
	// its own Access Attachment auto-sync side effect — see that
	// service's syncAccessAttachment doc comment for the full reasoning.
	// accessAttachmentSvc is passed twice, satisfying both the creator and
	// remover seams: Delete now also hard-deletes an active
	// AccessAttachment before its own record, the same service handling
	// both directions of that relationship. Its two foreign keys are
	// still Service and inventory.Device — deviceService (built above,
	// alongside deviceHandler) is reused rather than built fresh, for the
	// Device-status side effect ServiceEquipmentService.Create/Update also
	// carry: attaching/detaching a Device flips it Active/Unused (see
	// that service's own doc comment) — unless customerDeviceRepo still
	// shows it attached to a Customer directly, in which case losing its
	// Service leaves it Active, not Unused.
	serviceEquipmentSvc := serviceequipmentservice.NewServiceEquipmentService(
		serviceEquipmentRepo, deviceService, deviceService, customerDeviceRepo,
		onuAuthorizationSvc, accessInterfaceSvc, accessAttachmentSvc, accessAttachmentSvc)
	serviceEquipmentHandler := serviceequipmenthttpapi.NewServiceEquipmentHandler(serviceEquipmentSvc)

	// Kontron ONU authorization (internal/provisioning/kontron) is this
	// codebase's first real vendor-specific *write* command surface —
	// see that package's own doc comment on why it is a sibling of, not
	// nested inside, the read-only internal/diagnostics/kontron above.
	// It reuses kontronDialer rather than building a second Dialer: both
	// resolve the exact same "OLT ID -> live shell" question, just for
	// different commands run over the resulting shell.
	provisioningKontronSvc := provisioningkontronservice.NewAuthorizationService(kontronDialer, cfg.Kontron.ManagementServiceProfile)
	provisioningKontronHandler := provisioningkontronhttpapi.NewAuthorizationHandler(provisioningKontronSvc)

	// Authorize-and-create-Device (internal/provisioning/kontron/service.
	// AuthorizeAndCreateDeviceService) composes provisioningKontronSvc
	// (the OLT-side command) with deviceService and onuAuthorizationSvc
	// (both built above) rather than building any of them fresh — see
	// that service's own doc comment for why this exists as a single
	// action instead of two independently-forgettable ones. ponPortSvc and
	// accessInterfaceSvc (also both built above) back this service's own
	// Access Network topology sync — see its syncAccessTopology doc
	// comment.
	provisioningKontronAuthorizeAndCreateDeviceSvc := provisioningkontronservice.NewAuthorizeAndCreateDeviceService(
		provisioningKontronSvc, deviceService, onuAuthorizationSvc, ponPortSvc, accessInterfaceSvc, clock.New())
	provisioningKontronAuthorizeAndCreateDeviceHandler := provisioningkontronhttpapi.NewAuthorizeAndCreateDeviceHandler(provisioningKontronAuthorizeAndCreateDeviceSvc)

	// Access Topology (internal/accesstopology) resolves where a
	// Customer's equipment sits on the access network — the OLT and
	// interface a diagnostic needs — reusing accessAttachmentRepo,
	// accessInterfaceRepo, and ponPortRepo (the single-equipment
	// Resolver) plus locationRepo, serviceRepo, and serviceEquipmentRepo
	// (CustomerResolver's Customer -> Location -> Service ->
	// ServiceEquipment fan-out, one layer up). Like Kontron diagnostics
	// above, this has no repository or persistence of its own — it only
	// traverses relationships already stored by those six repositories
	// (see internal/accesstopology's own package doc comment).
	accessTopologyResolver := accesstopology.NewResolver(accessAttachmentRepo, accessInterfaceRepo, ponPortRepo)
	customerResolver := accesstopology.NewCustomerResolver(locationRepo, serviceRepo, serviceEquipmentRepo, accessTopologyResolver)
	accessTopologyHandler := accesstopologyhttpapi.NewAccessTopologyHandler(customerResolver)

	// Kontron's Plugin (internal/provisioning/kontron/plugin) is this
	// codebase's first real Plugin: applying (Provision/Resume) or
	// removing (Suspend/Disconnect) a Service's Product-specific Kontron
	// service-profile on its ONU, wired through internal/workflow's
	// Capability/Registry system rather than a standalone REST endpoint
	// (unlike provisioningKontronHandler above). It reuses kontronDialer,
	// oltRepo, oltModelRepo, accessTopologyResolver, and
	// provisioningProfileRepo — all already built above for their own
	// callers — rather than constructing anything new. See the
	// pluginRegistry comment above for how this registration interacts
	// with MockPlugin's.
	provisioningKontronServiceProfileSvc := provisioningkontronservice.NewServiceProfileService(
		kontronDialer, accessTopologyResolver, oltRepo, oltModelRepo, provisioningProfileRepo)
	pluginRegistry.Register(provisioningkontronplugin.New(provisioningKontronServiceProfileSvc))

	// Kontron ONU deauthorization ("Delete ONU", internal/provisioning/
	// kontron/service.DeauthorizationService) is a standalone REST
	// endpoint like provisioningKontronHandler above, not a workflow
	// Plugin: it is triggered from the Device Detail page (a DeviceID in
	// hand, nothing else), fully removes the ONU's base authorization,
	// and — once that succeeds — marks whichever record it resolved
	// through (ServiceEquipment/AccessAttachment, or — when there is
	// none — onuAuthorizationSvc's OnuAuthorization, built above)
	// removed/deauthorized, and the Device itself Retired (see
	// markDeviceRetired's own doc comment). It reuses kontronDialer,
	// serviceEquipmentRepo/serviceEquipmentSvc, accessTopologyResolver,
	// oltRepo, oltModelRepo, accessAttachmentRepo/accessAttachmentSvc,
	// onuAuthorizationSvc, and deviceService — all already built above
	// for their own callers.
	provisioningKontronDeauthorizationSvc := provisioningkontronservice.NewDeauthorizationService(
		kontronDialer, serviceEquipmentRepo, serviceEquipmentSvc, accessTopologyResolver,
		oltRepo, oltModelRepo, accessAttachmentRepo, accessAttachmentSvc,
		onuAuthorizationSvc, onuAuthorizationSvc,
		deviceService, deviceService, clock.New(), cfg.Kontron.ManagementServiceProfile)
	provisioningKontronDeauthorizationHandler := provisioningkontronhttpapi.NewDeauthorizationHandler(provisioningKontronDeauthorizationSvc)

	// Customer removal ("Remove Customer",
	// internal/customer/removal.RemovalService) cascades a Customer's own
	// removal down through its Locations, Services, and equipment — see
	// that package's own doc comment for why this is a status-transition
	// cascade (Archived/Inactive/Disconnected/RemovedAt), never a hard
	// delete. It reuses customerRepo, locationRepo, serviceRepo,
	// deviceRepo, serviceEquipmentRepo/serviceEquipmentSvc,
	// accessAttachmentRepo/accessAttachmentSvc, and the same
	// provisioningKontronServiceProfileSvc already built above for the
	// Plugin — nothing new beneath it.
	customerRemovalSvc := customerremoval.NewRemovalService(
		customerRepo, locationRepo, serviceRepo, deviceRepo,
		serviceEquipmentRepo, serviceEquipmentSvc, accessAttachmentRepo, accessAttachmentSvc,
		provisioningKontronServiceProfileSvc, clock.New())
	customerRemovalHandler := customerhttpapi.NewRemovalHandler(customerRemovalSvc)

	// tokenIssuer is shared by auth.Middleware (validates incoming tokens)
	// and LoginHandler (issues new ones): both need to agree on the same
	// secret and expiration, and a single instance is the simplest way to
	// guarantee that rather than constructing it twice from the same
	// config values.
	tokenIssuer := auth.NewTokenIssuer([]byte(cfg.JWT.Secret), cfg.JWT.Expiration, clock.New())

	// userRepo itself is built much earlier in this file (see the comment
	// above noteHandler's wiring) — authService, userManagementSvc,
	// profileSvc, and authzMiddleware below all reuse that exact
	// instance rather than each constructing their own, so every one of
	// them is always looking at the same table.
	authService := auth.NewAuthService(userRepo, tokenIssuer)
	loginHandler := authhttpapi.NewLoginHandler(authService, cfg.JWT.Expiration)

	userManagementSvc := authservice.NewUserManagementService(userRepo)
	userHandler := authhttpapi.NewUserHandler(userManagementSvc)

	// profileSvc/profileHandler back /api/v1/me: a signed-in caller
	// viewing and editing their own account (name, password), as opposed
	// to userManagementSvc/userHandler's Administrator-only /api/v1/users
	// (any account, by ID). See internal/auth/service.ProfileService's
	// own doc comment for why this is a separate type rather than more
	// methods on UserManagementService.
	profileSvc := authservice.NewProfileService(userRepo)
	profileHandler := authhttpapi.NewProfileHandler(profileSvc)

	authzMiddleware := authz.NewMiddleware(userRepo)

	router := api.NewRouter(api.Dependencies{
		Logger:                     logger,
		HealthCheckers:             healthCheckers,
		Version:                    version.Version,
		Commit:                     version.Commit,
		SiteHandler:                siteHandler,
		BuildingHandler:            buildingHandler,
		RoomHandler:                roomHandler,
		RackHandler:                rackHandler,
		DeviceHandler:              deviceHandler,
		DeviceManufacturerHandler:  deviceManufacturerHandler,
		DeviceModelHandler:         deviceModelHandler,
		CustomerHandler:            customerHandler,
		CustomerRemovalHandler:     customerRemovalHandler,
		LocationHandler:            locationHandler,
		ContactHandler:             contactHandler,
		CatalogHandler:             catalogHandler,
		ProductHandler:             productHandler,
		ProviderHandler:            providerHandler,
		ProvisioningProfileHandler: provisioningProfileHandler,
		ProvisioningKontronHandler: provisioningKontronHandler,
		ProvisioningKontronDeauthorizationHandler:          provisioningKontronDeauthorizationHandler,
		ProvisioningKontronAuthorizeAndCreateDeviceHandler: provisioningKontronAuthorizeAndCreateDeviceHandler,
		ServiceProfileHandler:                              serviceProfileHandler,
		DiagnosticsHandler:                                 diagnosticsHandler,
		KontronHandler:                                     kontronHandler,
		AccessTopologyHandler:                              accessTopologyHandler,
		ServiceHandler:                                     serviceHandler,
		ServiceEquipmentHandler:                            serviceEquipmentHandler,
		CustomerDeviceHandler:                              customerDeviceHandler,
		WorkflowHandler:                                    workflowHandler,
		EventHandler:                                       eventHandler,
		NoteHandler:                                        noteHandler,
		ReportHandler:                                      reportHandler,
		AccessNetworkHandler:                               accessNetworkHandler,
		OLTHandler:                                         oltHandler,
		OLTModelHandler:                                    oltModelHandler,
		PONPortHandler:                                     ponPortHandler,
		AccessInterfaceHandler:                             accessInterfaceHandler,
		AccessAttachmentHandler:                            accessAttachmentHandler,
		AuthenticationHandler:                              authenticationHandler,
		ConnectionProfileHandler:                           connectionProfileHandler,
		Tokens:                                             tokenIssuer,
		LoginHandler:                                       loginHandler,
		UserHandler:                                        userHandler,
		ProfileHandler:                                     profileHandler,
		Authz:                                              authzMiddleware,
		AllowedOrigin:                                      cfg.HTTP.AllowedOrigin,
	})

	srv := httpserver.New(httpserver.Config{
		Addr:            cfg.Addr(),
		ReadTimeout:     cfg.HTTP.ReadTimeout,
		WriteTimeout:    cfg.HTTP.WriteTimeout,
		IdleTimeout:     cfg.HTTP.IdleTimeout,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}, router, logger)

	// The workflow worker runs alongside the HTTP server, sharing the same
	// signal-driven ctx: Worker.Run returns as soon as ctx is cancelled,
	// the same graceful-shutdown trigger httpserver.Server.Run reacts to.
	// workerDone is waited on below so "palladium server stopped cleanly"
	// is only logged once the worker has actually finished its current
	// poll iteration, not merely asked to stop.
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		workflowWorker.Run(ctx)
	}()

	if err := srv.Run(ctx); err != nil {
		return err
	}
	<-workerDone

	logger.Info("palladium server stopped cleanly")
	return nil
}
