// Package api assembles the HTTP router: middleware, health endpoints, and
// the versioned API route group that later phases will populate. It
// contains no business logic.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	accessattachmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessattachment/httpapi"
	accessinterfacehttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessinterface/httpapi"
	accessnetworkhttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authhttpapi "github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	cataloghttpapi "github.com/paladindigitalgh/palladium-oss/internal/catalog/httpapi"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	diagnosticshttpapi "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/health"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	locationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
	olthttpapi "github.com/paladindigitalgh/palladium-oss/internal/olt/httpapi"
	ponporthttpapi "github.com/paladindigitalgh/palladium-oss/internal/ponport/httpapi"
	producthttpapi "github.com/paladindigitalgh/palladium-oss/internal/product/httpapi"
	provisioninghttpapi "github.com/paladindigitalgh/palladium-oss/internal/provisioning/httpapi"
	servicehttpapi "github.com/paladindigitalgh/palladium-oss/internal/service/httpapi"
	serviceequipmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/httpapi"
	serviceprofilehttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/httpapi"
)

// Dependencies holds everything the router needs to wire up routes and
// middleware. It is the composition root's single point of contact with
// this package, keeping construction explicit rather than relying on
// globals or a framework-managed container.
type Dependencies struct {
	Logger                  *slog.Logger
	HealthCheckers          []health.Checker
	Version                 string
	Commit                  string
	SiteHandler             *httpapi.SiteHandler
	CustomerHandler         *customerhttpapi.CustomerHandler
	LocationHandler         *locationhttpapi.LocationHandler
	CatalogHandler          *cataloghttpapi.CatalogHandler
	ProductHandler          *producthttpapi.ProductHandler
	ServiceHandler          *servicehttpapi.ServiceHandler
	ServiceEquipmentHandler *serviceequipmenthttpapi.ServiceEquipmentHandler
	ProvisioningHandler     *provisioninghttpapi.ProvisioningHandler
	AccessNetworkHandler    *accessnetworkhttpapi.AccessNetworkHandler
	OLTHandler              *olthttpapi.OLTHandler
	PONPortHandler          *ponporthttpapi.PONPortHandler
	AccessInterfaceHandler  *accessinterfacehttpapi.AccessInterfaceHandler
	AccessAttachmentHandler *accessattachmenthttpapi.AccessAttachmentHandler
	ServiceProfileHandler   *serviceprofilehttpapi.ServiceProfileHandler
	DiagnosticsHandler      *diagnosticshttpapi.DiagnosticsHandler
	Tokens                  *auth.TokenIssuer
	LoginHandler            *authhttpapi.LoginHandler
	Authz                   *authz.Middleware
}

// NewRouter builds the application's http.Handler.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(RequestLogger(deps.Logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	healthHandler := health.NewHandler(deps.HealthCheckers, deps.Version, deps.Commit)
	r.Get("/healthz", healthHandler.Live)
	r.Get("/readyz", healthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		// Domain routes (customers, services, workflows, ...) are mounted
		// here as those phases are implemented.

		inventoryHandler := inventory.NewHandler()
		r.Route("/inventory", func(r chi.Router) {
			// Temporary: verifies the inventory domain is wired correctly.
			// Replace with real CRUD routes once the repository layer has
			// a SQL implementation.
			r.Get("/schema", inventoryHandler.Schema)
		})

		// /auth/login is deliberately outside the auth.Middleware group
		// below: a caller has no token yet at the point they are trying to
		// obtain one. No other /auth route exists (no logout, no refresh,
		// no password reset — see this milestone's scope), so there is
		// nothing else here that would need to distinguish authenticated
		// from unauthenticated.
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", deps.LoginHandler.Login)
		})

		// /sites is the first authenticated resource: every route in this
		// group requires a valid JWT (see auth.Middleware), applied once
		// for the whole group so the token is validated exactly once per
		// request regardless of which sub-group below handles it — see
		// authz.Middleware's doc comment on why that must run after this,
		// not instead of it. Building, Room, Rack, and Device follow the
		// same shape once their own handlers exist — deliberately not
		// implemented yet (see this milestone's scope).
		//
		// Read and write split into two sub-groups because they require
		// different capabilities (goal 4): GET is available to every role
		// (RequireInventoryRead), while POST/PUT/DELETE require
		// RequireInventoryWrite, which Viewer does not have. No group here
		// requires Administrator exclusively — Operator satisfies both.
		r.Route("/sites", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryRead())
				r.Get("/", deps.SiteHandler.List)
				r.Get("/{id}", deps.SiteHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryWrite())
				r.Post("/", deps.SiteHandler.Create)
				r.Put("/{id}", deps.SiteHandler.Update)
				r.Delete("/{id}", deps.SiteHandler.Delete)
			})
		})

		// /customers uses the exact same shape as /sites above, with the
		// exact same authorization model (goal 6) applied via its own
		// named capabilities (RequireCustomerRead/RequireCustomerWrite,
		// not a reuse of RequireInventoryRead/RequireInventoryWrite — see
		// authz.CanReadCustomers's doc comment for why Customers and
		// Inventory each get their own capability even though the role
		// rules are identical today).
		r.Route("/customers", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireCustomerRead())
				r.Get("/", deps.CustomerHandler.List)
				r.Get("/{id}", deps.CustomerHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireCustomerWrite())
				r.Post("/", deps.CustomerHandler.Create)
				r.Put("/{id}", deps.CustomerHandler.Update)
				r.Delete("/{id}", deps.CustomerHandler.Delete)
			})
		})

		// /locations uses the exact same shape as /customers above, with
		// the exact same authorization model ("match Customer permissions")
		// applied via its own named capabilities
		// (RequireLocationRead/RequireLocationWrite, not a reuse of
		// RequireCustomerRead/RequireCustomerWrite — see
		// authz.CanReadLocations's doc comment for why Locations and
		// Customers each get their own capability even though the role
		// rules are identical today).
		r.Route("/locations", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireLocationRead())
				r.Get("/", deps.LocationHandler.List)
				r.Get("/{id}", deps.LocationHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireLocationWrite())
				r.Post("/", deps.LocationHandler.Create)
				r.Put("/{id}", deps.LocationHandler.Update)
				r.Delete("/{id}", deps.LocationHandler.Delete)
			})
		})

		// /catalogs and /products share one capability pair
		// (RequireCatalogRead/RequireCatalogWrite), unlike every other pair
		// of domains mounted above: a Product only exists nested inside a
		// ProductCatalog (see product.Product's required CatalogID), so
		// "who can read/write the catalog" and "who can read/write a
		// product in it" are the same question asked at two levels of one
		// domain, not two domains that happen to share a rule today (see
		// authz.CanReadCatalog's doc comment). Each still gets its own
		// route group — the capability is shared, the routing is not.
		r.Route("/catalogs", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireCatalogRead())
				r.Get("/", deps.CatalogHandler.List)
				r.Get("/{id}", deps.CatalogHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireCatalogWrite())
				r.Post("/", deps.CatalogHandler.Create)
				r.Put("/{id}", deps.CatalogHandler.Update)
				r.Delete("/{id}", deps.CatalogHandler.Delete)
			})
		})

		r.Route("/products", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireCatalogRead())
				r.Get("/", deps.ProductHandler.List)
				r.Get("/{id}", deps.ProductHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireCatalogWrite())
				r.Post("/", deps.ProductHandler.Create)
				r.Put("/{id}", deps.ProductHandler.Update)
				r.Delete("/{id}", deps.ProductHandler.Delete)
			})
		})

		// /services gets its own dedicated capability pair
		// (RequireServiceRead/RequireServiceWrite), not a reuse of
		// /catalogs' or /locations' — unlike Catalog and Product, a
		// Service is not "part of" a Location or a Product, it merely
		// references both (see authz.CanReadServices's doc comment).
		r.Route("/services", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireServiceRead())
				r.Get("/", deps.ServiceHandler.List)
				r.Get("/{id}", deps.ServiceHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireServiceWrite())
				r.Post("/", deps.ServiceHandler.Create)
				r.Put("/{id}", deps.ServiceHandler.Update)
				r.Delete("/{id}", deps.ServiceHandler.Delete)
			})
		})

		// /service-equipment gets its own dedicated capability pair
		// (RequireServiceEquipmentRead/RequireServiceEquipmentWrite), not a
		// reuse of /services' — per this milestone's explicit instruction
		// ("do not reuse Service capabilities"; see
		// authz.CanReadServiceEquipment's doc comment for why).
		r.Route("/service-equipment", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireServiceEquipmentRead())
				r.Get("/", deps.ServiceEquipmentHandler.List)
				r.Get("/{id}", deps.ServiceEquipmentHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireServiceEquipmentWrite())
				r.Post("/", deps.ServiceEquipmentHandler.Create)
				r.Put("/{id}", deps.ServiceEquipmentHandler.Update)
				r.Delete("/{id}", deps.ServiceEquipmentHandler.Delete)
			})
		})

		// /provisioning-jobs gets its own dedicated capability pair
		// (RequireProvisioningRead/RequireProvisioningWrite), not a reuse
		// of /services' — per this milestone's explicit instruction ("do
		// not reuse Service permissions"; see
		// authz.CanReadProvisioning's doc comment for why). The write
		// group also covers the state-machine action sub-routes
		// (start/succeed/fail/cancel/retry): driving a transition is a
		// write, the same as create/delete, and every one of those
		// actions requires exactly the same capability — there is no
		// narrower permission this milestone asks for (e.g. "can cancel
		// but not start"), so splitting them into further sub-groups
		// would add structure with no corresponding rule to justify it.
		r.Route("/provisioning-jobs", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireProvisioningRead())
				r.Get("/", deps.ProvisioningHandler.List)
				r.Get("/{id}", deps.ProvisioningHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireProvisioningWrite())
				r.Post("/", deps.ProvisioningHandler.Create)
				r.Delete("/{id}", deps.ProvisioningHandler.Delete)
				r.Post("/{id}/start", deps.ProvisioningHandler.Start)
				r.Post("/{id}/succeed", deps.ProvisioningHandler.Succeed)
				r.Post("/{id}/fail", deps.ProvisioningHandler.Fail)
				r.Post("/{id}/cancel", deps.ProvisioningHandler.Cancel)
				r.Post("/{id}/retry", deps.ProvisioningHandler.Retry)
			})
		})

		// /access-networks, /olts, and /pon-ports share one capability
		// pair (RequireAccessNetworkRead/RequireAccessNetworkWrite),
		// mirroring /catalogs and /products above: an OLT only exists
		// nested inside an AccessNetwork, and a PONPort only exists
		// nested inside an OLT (see authz.CanReadAccessNetwork's doc
		// comment), so "who can read/write" each of the three is the same
		// question asked at three levels of one domain, not three domains
		// that happen to share a rule today. Each still gets its own
		// route group — the capability is shared, the routing is not.
		r.Route("/access-networks", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessNetworkRead())
				r.Get("/", deps.AccessNetworkHandler.List)
				r.Get("/{id}", deps.AccessNetworkHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessNetworkWrite())
				r.Post("/", deps.AccessNetworkHandler.Create)
				r.Put("/{id}", deps.AccessNetworkHandler.Update)
				r.Delete("/{id}", deps.AccessNetworkHandler.Delete)
			})
		})

		r.Route("/olts", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessNetworkRead())
				r.Get("/", deps.OLTHandler.List)
				r.Get("/{id}", deps.OLTHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessNetworkWrite())
				r.Post("/", deps.OLTHandler.Create)
				r.Put("/{id}", deps.OLTHandler.Update)
				r.Delete("/{id}", deps.OLTHandler.Delete)
			})
		})

		r.Route("/pon-ports", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessNetworkRead())
				r.Get("/", deps.PONPortHandler.List)
				r.Get("/{id}", deps.PONPortHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessNetworkWrite())
				r.Post("/", deps.PONPortHandler.Create)
				r.Put("/{id}", deps.PONPortHandler.Update)
				r.Delete("/{id}", deps.PONPortHandler.Delete)
			})
		})

		// /access-interfaces and /access-attachments share one capability
		// pair (RequireAccessTopologyRead/RequireAccessTopologyWrite),
		// deliberately distinct from RequireAccessNetworkRead/Write above
		// (see authz.CanReadAccessTopology's doc comment): an
		// AccessAttachment only exists nested inside an AccessInterface,
		// so "who can read/write" each of the two is the same question
		// asked at two levels of one domain, not two domains that happen
		// to share a rule today. Each still gets its own route group —
		// the capability is shared, the routing is not.
		r.Route("/access-interfaces", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessTopologyRead())
				r.Get("/", deps.AccessInterfaceHandler.List)
				r.Get("/{id}", deps.AccessInterfaceHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessTopologyWrite())
				r.Post("/", deps.AccessInterfaceHandler.Create)
				r.Put("/{id}", deps.AccessInterfaceHandler.Update)
				r.Delete("/{id}", deps.AccessInterfaceHandler.Delete)
			})
		})

		r.Route("/access-attachments", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessTopologyRead())
				r.Get("/", deps.AccessAttachmentHandler.List)
				r.Get("/{id}", deps.AccessAttachmentHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAccessTopologyWrite())
				r.Post("/", deps.AccessAttachmentHandler.Create)
				r.Put("/{id}", deps.AccessAttachmentHandler.Update)
				r.Delete("/{id}", deps.AccessAttachmentHandler.Delete)
			})
		})

		// /service-profiles gets its own dedicated capability pair
		// (RequireServiceProfilesRead/RequireServiceProfilesWrite), not a
		// reuse of /catalogs' or /products' — per this milestone's
		// explicit instruction ("do not reuse Product permissions"). A
		// Service references both a Product and a ServiceProfile, but
		// they represent different business concepts that may diverge in
		// authorization requirements later (see
		// authz.CanReadServiceProfiles's doc comment).
		r.Route("/service-profiles", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireServiceProfilesRead())
				r.Get("/", deps.ServiceProfileHandler.List)
				r.Get("/{id}", deps.ServiceProfileHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireServiceProfilesWrite())
				r.Post("/", deps.ServiceProfileHandler.Create)
				r.Put("/{id}", deps.ServiceProfileHandler.Update)
				r.Delete("/{id}", deps.ServiceProfileHandler.Delete)
			})
		})

		// /diagnostics has no read/write split — RequireDiagnostics is
		// the one capability guarding this whole route (see
		// authz.CanRunDiagnostics's doc comment for why running a
		// diagnostic does not decompose into a Read/Write pair the way
		// every other domain mounted above does).
		r.Route("/diagnostics", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireDiagnostics())
				r.Post("/basic-onu-check", deps.DiagnosticsHandler.BasicONUCheck)
			})
		})
	})

	return r
}
