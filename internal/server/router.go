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

	"github.com/paladindigitalgh/palladium-oss/internal/health"
)

// Dependencies holds everything the router needs to wire up routes and
// middleware. It is the composition root's single point of contact with
// this package, keeping construction explicit rather than relying on
// globals or a framework-managed container.
type Dependencies struct {
	Logger         *slog.Logger
	HealthCheckers []health.Checker
	Version        string
	Commit         string
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
		// Domain routes (inventory, customers, services, workflows, ...)
		// are mounted here as those phases are implemented.
	})

	return r
}
