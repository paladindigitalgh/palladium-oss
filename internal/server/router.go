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
	authenticationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/authentication/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	cataloghttpapi "github.com/paladindigitalgh/palladium-oss/internal/catalog/httpapi"
	connectionprofilehttpapi "github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/httpapi"
	contacthttpapi "github.com/paladindigitalgh/palladium-oss/internal/contact/httpapi"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	diagnosticshttpapi "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/httpapi"
	eventhttpapi "github.com/paladindigitalgh/palladium-oss/internal/event/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/health"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	locationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
	olthttpapi "github.com/paladindigitalgh/palladium-oss/internal/olt/httpapi"
	ponporthttpapi "github.com/paladindigitalgh/palladium-oss/internal/ponport/httpapi"
	producthttpapi "github.com/paladindigitalgh/palladium-oss/internal/product/httpapi"
	servicehttpapi "github.com/paladindigitalgh/palladium-oss/internal/service/httpapi"
	serviceequipmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/httpapi"
	serviceprofilehttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/httpapi"
	workflowhttpapi "github.com/paladindigitalgh/palladium-oss/internal/workflow/httpapi"
)

// Dependencies holds everything the router needs to wire up routes and
// middleware. It is the composition root's single point of contact with
// this package, keeping construction explicit rather than relying on
// globals or a framework-managed container.
type Dependencies struct {
	Logger                   *slog.Logger
	HealthCheckers           []health.Checker
	Version                  string
	Commit                   string
	SiteHandler              *httpapi.SiteHandler
	BuildingHandler          *httpapi.BuildingHandler
	RoomHandler              *httpapi.RoomHandler
	DeviceHandler            *httpapi.DeviceHandler
	CustomerHandler          *customerhttpapi.CustomerHandler
	LocationHandler          *locationhttpapi.LocationHandler
	ContactHandler           *contacthttpapi.ContactHandler
	CatalogHandler           *cataloghttpapi.CatalogHandler
	ProductHandler           *producthttpapi.ProductHandler
	ServiceHandler           *servicehttpapi.ServiceHandler
	ServiceEquipmentHandler  *serviceequipmenthttpapi.ServiceEquipmentHandler
	WorkflowHandler          *workflowhttpapi.WorkflowHandler
	EventHandler             *eventhttpapi.EventHandler
	AccessNetworkHandler     *accessnetworkhttpapi.AccessNetworkHandler
	OLTHandler               *olthttpapi.OLTHandler
	PONPortHandler           *ponporthttpapi.PONPortHandler
	AccessInterfaceHandler   *accessinterfacehttpapi.AccessInterfaceHandler
	AccessAttachmentHandler  *accessattachmenthttpapi.AccessAttachmentHandler
	ServiceProfileHandler    *serviceprofilehttpapi.ServiceProfileHandler
	DiagnosticsHandler       *diagnosticshttpapi.DiagnosticsHandler
	AuthenticationHandler    *authenticationhttpapi.AuthenticationHandler
	ConnectionProfileHandler *connectionprofilehttpapi.ConnectionProfileHandler
	Tokens                   *auth.TokenIssuer
	LoginHandler             *authhttpapi.LoginHandler
	Authz                    *authz.Middleware
	// AllowedOrigin is the frontend origin CORS middleware accepts
	// cross-origin requests from (see corsMiddleware). Empty disables
	// CORS headers entirely, which is fine for tests that never go
	// through a browser.
	AllowedOrigin string
}

// NewRouter builds the application's http.Handler.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(RequestLogger(deps.Logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	if deps.AllowedOrigin != "" {
		r.Use(corsMiddleware(deps.AllowedOrigin))
	}

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

		// /buildings and /rooms use the exact same shape as /sites above,
		// reusing the same RequireInventoryRead/RequireInventoryWrite
		// capabilities — Building and Room are Inventory, the same
		// hierarchy Site and Device belong to, not domains of their own.
		r.Route("/buildings", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryRead())
				r.Get("/", deps.BuildingHandler.List)
				r.Get("/{id}", deps.BuildingHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryWrite())
				r.Post("/", deps.BuildingHandler.Create)
				r.Put("/{id}", deps.BuildingHandler.Update)
				r.Delete("/{id}", deps.BuildingHandler.Delete)
			})
		})

		r.Route("/rooms", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryRead())
				r.Get("/", deps.RoomHandler.List)
				r.Get("/{id}", deps.RoomHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryWrite())
				r.Post("/", deps.RoomHandler.Create)
				r.Put("/{id}", deps.RoomHandler.Update)
				r.Delete("/{id}", deps.RoomHandler.Delete)
			})
		})

		// /devices uses the exact same shape as /sites above, reusing the
		// same RequireInventoryRead/RequireInventoryWrite capabilities —
		// Device is Inventory, the same as Site, not a domain of its own.
		r.Route("/devices", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryRead())
				r.Get("/", deps.DeviceHandler.List)
				r.Get("/{id}", deps.DeviceHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireInventoryWrite())
				r.Post("/", deps.DeviceHandler.Create)
				r.Put("/{id}", deps.DeviceHandler.Update)
				r.Delete("/{id}", deps.DeviceHandler.Delete)
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

		// /contacts uses the exact same shape as /locations above, with its
		// own dedicated capability pair (RequireContactRead/
		// RequireContactWrite) rather than reusing Location's or Customer's
		// -- the same reasoning CanReadContacts's doc comment gives.
		r.Route("/contacts", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireContactRead())
				r.Get("/", deps.ContactHandler.List)
				r.Get("/{id}", deps.ContactHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireContactWrite())
				r.Post("/", deps.ContactHandler.Create)
				r.Put("/{id}", deps.ContactHandler.Update)
				r.Delete("/{id}", deps.ContactHandler.Delete)
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

		// /workflow-instances gets its own dedicated capability pair
		// (RequireWorkflowRead/RequireWorkflowWrite), not a reuse of
		// /services'. The write group also covers the action sub-routes
		// (execute/cancel/retry): driving execution or a transition is a
		// write, the same as create/delete.
		r.Route("/workflow-instances", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireWorkflowRead())
				r.Get("/", deps.WorkflowHandler.List)
				r.Get("/{id}", deps.WorkflowHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireWorkflowWrite())
				r.Post("/", deps.WorkflowHandler.Create)
				r.Delete("/{id}", deps.WorkflowHandler.Delete)
				r.Post("/{id}/execute", deps.WorkflowHandler.Execute)
				r.Post("/{id}/cancel", deps.WorkflowHandler.Cancel)
				r.Post("/{id}/retry", deps.WorkflowHandler.Retry)
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

		// /events is read-only: there is no write route, since events are
		// written internally by domain/workflow code, never posted by a
		// client (see internal/event/httpapi's package doc comment).
		r.Route("/events", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))
			r.Use(deps.Authz.RequireEventRead())
			r.Get("/", deps.EventHandler.List)
			r.Get("/recent", deps.EventHandler.ListRecent)
		})

		// /authentication-methods gets its own dedicated capability pair
		// (RequireAuthenticationRead/RequireAuthenticationWrite), not a
		// reuse of any other domain's — per this milestone's explicit
		// goal 5, naming these two functions independently of Connection
		// Profile's own pair below (see authz.CanReadAuthentication's doc
		// comment for why, even though a ConnectionProfile references an
		// Authentication record by ID).
		r.Route("/authentication-methods", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAuthenticationRead())
				r.Get("/", deps.AuthenticationHandler.List)
				r.Get("/{id}", deps.AuthenticationHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireAuthenticationWrite())
				r.Post("/", deps.AuthenticationHandler.Create)
				r.Put("/{id}", deps.AuthenticationHandler.Update)
				r.Delete("/{id}", deps.AuthenticationHandler.Delete)
			})
		})

		// /connection-profiles gets its own dedicated capability pair
		// (RequireConnectionProfilesRead/RequireConnectionProfilesWrite),
		// not a reuse of /authentication-methods' — per this milestone's
		// explicit goal 5 (see authz.CanReadConnectionProfiles's doc
		// comment for why).
		r.Route("/connection-profiles", func(r chi.Router) {
			r.Use(auth.Middleware(deps.Tokens))

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireConnectionProfilesRead())
				r.Get("/", deps.ConnectionProfileHandler.List)
				r.Get("/{id}", deps.ConnectionProfileHandler.Get)
			})

			r.Group(func(r chi.Router) {
				r.Use(deps.Authz.RequireConnectionProfilesWrite())
				r.Post("/", deps.ConnectionProfileHandler.Create)
				r.Put("/{id}", deps.ConnectionProfileHandler.Update)
				r.Delete("/{id}", deps.ConnectionProfileHandler.Delete)
			})
		})
	})

	return r
}

// corsMiddleware sets the CORS headers a browser requires to accept a
// cross-origin response, and short-circuits the CORS preflight OPTIONS
// request every non-trivial cross-origin call triggers (any request
// carrying an Authorization header, which is every authenticated route
// here) before it ever reaches auth.Middleware -- a preflight request
// carries no Authorization header itself, so letting it fall through to
// authentication would reject every preflight and break the real
// request that follows it.
func corsMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
