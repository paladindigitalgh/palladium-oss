// Package service is the Diagnostics framework's business logic layer.
// It sits between the HTTP layer and the Registry: HTTP handlers never
// look up a Diagnostic directly (see internal/diagnostics/httpapi), and
// the Registry never knows anything about HTTP — this is where those two
// responsibilities meet. It mirrors every other domain's service layer
// in this codebase in spirit (validate/locate, then delegate, with no
// business logic of its own), even though goal 6 is explicit that this
// one has no persistence: "no background jobs, no database writes."
package service

import (
	"context"
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// DiagnosticsService is the Diagnostics framework's business logic.
//
// It depends only on diagnostics.Registry — not a repository, because
// there is none, and not clock.Clock, because every domain in this
// codebase only injects a clock to timestamp something it persists, and
// this service persists nothing; the timestamps in a diagnostics.Result
// are the Diagnostic's own concern (see BasicONUCheck.Run), not this
// service's.
type DiagnosticsService struct {
	registry diagnostics.Registry
}

// NewDiagnosticsService builds a DiagnosticsService.
func NewDiagnosticsService(registry diagnostics.Registry) *DiagnosticsService {
	return &DiagnosticsService{registry: registry}
}

// Run looks up the Diagnostic registered under name, executes it with
// request, and returns its Result. An unrecognized name is reported as
// an apperror.KindNotFound error — the same "not found" vocabulary every
// other domain in this codebase uses for "the thing you asked for by
// identity does not exist" — rather than, say, a KindInvalid error: name
// is not malformed input, it correctly names a Diagnostic that simply
// is not registered (a deployment/configuration gap, not a caller
// mistake in the general case).
func (s *DiagnosticsService) Run(ctx context.Context, name string, request diagnostics.Request) (*diagnostics.Result, error) {
	diagnostic, ok := s.registry.Get(name)
	if !ok {
		return nil, apperror.NotFound(fmt.Sprintf("diagnostic %q not found", name))
	}
	return diagnostic.Run(ctx, request)
}
