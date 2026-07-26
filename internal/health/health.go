// Package health exposes liveness and readiness HTTP handlers. It contains
// no business logic: dependency-specific checks are supplied by callers as
// Checker implementations (e.g. a database ping added in a later phase).
package health

import (
	"context"
	"encoding/json"
	"net/http"
)

// Checker reports whether a dependency is currently healthy.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// Handler serves liveness and readiness endpoints.
type Handler struct {
	checkers []Checker
	version  string
	commit   string
}

// NewHandler builds a Handler. checkers are evaluated on every readiness
// request; an empty slice means the service is always ready.
func NewHandler(checkers []Checker, version, commit string) *Handler {
	return &Handler{checkers: checkers, version: version, commit: commit}
}

// Live reports that the process is running. It never checks dependencies.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": h.version,
		"commit":  h.commit,
	})
}

// Ready reports whether the service is ready to accept traffic by running
// every registered Checker.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string, len(h.checkers))
	status := http.StatusOK

	for _, checker := range h.checkers {
		if err := checker.Check(r.Context()); err != nil {
			checks[checker.Name()] = err.Error()
			status = http.StatusServiceUnavailable
			continue
		}
		checks[checker.Name()] = "ok"
	}

	body := map[string]any{
		"status": "ready",
		"checks": checks,
	}
	if status != http.StatusOK {
		body["status"] = "unavailable"
	}

	writeJSON(w, status, body)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
