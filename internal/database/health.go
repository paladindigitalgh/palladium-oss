package database

import "context"

// HealthChecker adapts a Pool to callers that expect a Name()/Check(ctx)
// pair (see internal/health.Checker). It is defined here rather than by
// importing internal/health, so the persistence layer has no dependency on
// the HTTP layer — Go's structural typing lets the health package accept it
// without either side knowing about the other.
type HealthChecker struct {
	pool *Pool
}

// NewHealthChecker builds a HealthChecker for pool.
func NewHealthChecker(pool *Pool) *HealthChecker {
	return &HealthChecker{pool: pool}
}

// Name identifies this checker in readiness output.
func (h *HealthChecker) Name() string {
	return "database"
}

// Check pings the pool.
func (h *HealthChecker) Check(ctx context.Context) error {
	return h.pool.Ping(ctx)
}
