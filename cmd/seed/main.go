// Command seed populates a freshly migrated database with a minimal,
// realistic demo dataset — one Customer, Location, Catalog, Provider,
// Product, Service Profile, Service, inventory Device, and Service
// Equipment assignment — so a new installation has something real to
// look at and act on (a Service the Workflow Engine can actually
// provision, suspend, and resume against internal/plugin/mock's
// simulated vendor) before any real customer data or hardware exists.
//
// It is a separate binary from cmd/bootstrap for the same reason
// cmd/bootstrap is separate from cmd/migrate: account creation, schema
// migration, and demo data are three independent, one-time installation
// concerns, not steps of a single tool. It is idempotent — running it
// again after demo data already exists does nothing — so it is safe to
// include in a repeatable local setup script.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	catalogpostgres "github.com/paladindigitalgh/palladium-oss/internal/catalog/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/config"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerpostgres "github.com/paladindigitalgh/palladium-oss/internal/customer/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	inventorypostgres "github.com/paladindigitalgh/palladium-oss/internal/inventory/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	locationpostgres "github.com/paladindigitalgh/palladium-oss/internal/location/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	productpostgres "github.com/paladindigitalgh/palladium-oss/internal/product/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
	providerpostgres "github.com/paladindigitalgh/palladium-oss/internal/provider/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	servicepostgres "github.com/paladindigitalgh/palladium-oss/internal/service/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	serviceequipmentpostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
	serviceprofilepostgres "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/postgres"
)

// demoCustomerName identifies the seeded Customer, both as the record's
// Name and as the marker Run checks for to decide whether demo data
// already exists.
const demoCustomerName = "Demo Fiber Customer"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	customers := customerpostgres.NewCustomerRepository(pool, clock.New(), id.New())
	locations := locationpostgres.NewLocationRepository(pool, clock.New(), id.New())
	catalogs := catalogpostgres.NewCatalogRepository(pool, clock.New(), id.New())
	providers := providerpostgres.NewProviderRepository(pool, clock.New(), id.New())
	products := productpostgres.NewProductRepository(pool, clock.New(), id.New())
	serviceProfiles := serviceprofilepostgres.NewServiceProfileRepository(pool, clock.New(), id.New())
	services := servicepostgres.NewServiceRepository(pool, clock.New(), id.New())
	devices := inventorypostgres.NewDeviceRepository(pool, clock.New(), id.New())
	serviceEquipment := serviceequipmentpostgres.NewServiceEquipmentRepository(pool, clock.New(), id.New())

	existing, err := customers.List(ctx)
	if err != nil {
		return fmt.Errorf("list customers: %w", err)
	}
	for _, c := range existing {
		if c.Name == demoCustomerName {
			fmt.Println("demo data already exists; nothing to do")
			return nil
		}
	}

	demoCustomer, err := customers.Create(ctx, customer.Customer{
		Name:         demoCustomerName,
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
	})
	if err != nil {
		return fmt.Errorf("create demo customer: %w", err)
	}

	demoLocation, err := locations.Create(ctx, location.Location{
		CustomerID: demoCustomer.ID,
		Name:       "Demo Service Address",
		Type:       location.LocationTypeService,
		Status:     location.LocationStatusActive,
		Address1:   "123 Main Street",
		City:       "Springfield",
		State:      "IL",
		PostalCode: "62704",
		Country:    "US",
	})
	if err != nil {
		return fmt.Errorf("create demo location: %w", err)
	}

	demoCatalog, err := catalogs.Create(ctx, catalog.ProductCatalog{
		Name:   "Residential Internet",
		Status: catalog.CatalogStatusActive,
	})
	if err != nil {
		return fmt.Errorf("create demo catalog: %w", err)
	}

	demoProvider, err := providers.Create(ctx, provider.Provider{
		Name:   "Demo Provider",
		Status: provider.StatusActive,
	})
	if err != nil {
		return fmt.Errorf("create demo provider: %w", err)
	}

	demoProduct, err := products.Create(ctx, product.Product{
		CatalogID:  demoCatalog.ID,
		ProviderID: demoProvider.ID,
		Name:       "Fiber 500/500",
		Category:   product.ProductCategoryInternet,
		Status:     product.ProductStatusActive,
	})
	if err != nil {
		return fmt.Errorf("create demo product: %w", err)
	}

	demoServiceProfile, err := serviceProfiles.Create(ctx, serviceprofile.ServiceProfile{
		Name:   "Residential Standard",
		Status: serviceprofile.StatusActive,
	})
	if err != nil {
		return fmt.Errorf("create demo service profile: %w", err)
	}

	demoService, err := services.Create(ctx, service.Service{
		LocationID:       demoLocation.ID,
		ProductID:        demoProduct.ID,
		ServiceProfileID: demoServiceProfile.ID,
		Status:           service.ServiceStatusActive,
		Description:      "Demo service for exercising the Workflow Engine without real hardware",
	})
	if err != nil {
		return fmt.Errorf("create demo service: %w", err)
	}

	demoDevice, err := devices.Create(ctx, inventory.Device{
		Metadata: inventory.Metadata{
			Name:        "Demo ONT",
			Description: "Simulated ONT — acted on by internal/plugin/mock, not real hardware",
		},
		Manufacturer: "Simulated Vendor",
		Model:        "ONT-100",
		SerialNumber: "SIM-ONT-0001",
		Status:       inventory.DeviceStatusInstalled,
	})
	if err != nil {
		return fmt.Errorf("create demo device: %w", err)
	}

	demoAssignment, err := serviceEquipment.Create(ctx, serviceequipment.ServiceEquipment{
		ServiceID:   demoService.ID,
		DeviceID:    demoDevice.ID,
		Role:        serviceequipment.EquipmentRoleONU,
		Description: "Demo ONU assignment",
	})
	if err != nil {
		return fmt.Errorf("create demo service equipment: %w", err)
	}

	fmt.Println("demo data created:")
	fmt.Printf("  customer:          %s (%s)\n", demoCustomer.Name, demoCustomer.ID)
	fmt.Printf("  service:           %s\n", demoService.ID)
	fmt.Printf("  device:            %s (%s)\n", demoDevice.Name, demoDevice.ID)
	fmt.Printf("  service equipment: %s\n", demoAssignment.ID)
	return nil
}
