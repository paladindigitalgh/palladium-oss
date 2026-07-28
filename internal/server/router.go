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

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authhttpapi "github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	cataloghttpapi "github.com/paladindigitalgh/palladium-oss/internal/catalog/httpapi"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/health"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	locationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
	producthttpapi "github.com/paladindigitalgh/palladium-oss/internal/product/httpapi"
)

// Dependencies holds everything the router needs to wire up routes and
// middleware. It is the composition root's single point of contact with
// this package, keeping construction explicit rather than relying on
// globals or a framework-managed container.
type Dependencies struct {
	Logger          *slog.Logger
	HealthCheckers  []health.Checker
	Version         string
	Commit          string
	SiteHandler     *httpapi.SiteHandler
	CustomerHandler *customerhttpapi.CustomerHandler
	LocationHandler *locationhttpapi.LocationHandler
	CatalogHandler  *cataloghttpapi.CatalogHandler
	ProductHandler  *producthttpapi.ProductHandler
	Tokens          *auth.TokenIssuer
	LoginHandler    *authhttpapi.LoginHandler
	Authz           *authz.Middleware
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
	})

	return r
}
