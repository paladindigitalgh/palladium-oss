// Package service is the Kontron diagnostics caller: it ties
// internal/olt/connect.Dialer (resolve an OLT by ID into a live shell)
// together with internal/diagnostics/kontron.Client (run one named
// command over that shell), always closing the connection afterward
// regardless of the command's own outcome. It is the one place those
// two packages actually get used together end to end.
//
// This mirrors every other domain's service layer in this codebase in
// shape (a thin layer between HTTP and the packages doing the real
// work), but has no repository of its own and no CRUD: there is nothing
// to persist here (see internal/diagnostics/kontron's own doc comment —
// "no wiring into internal/diagnostics's Diagnostic/Registry/HTTP
// framework" — this package is that framework's own narrower, purpose-
// built replacement for the Kontron case specifically).
package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
)

// dialer is the seam KontronService depends on instead of the concrete
// *connect.Dialer — the same narrowing pattern
// internal/workflow/engine.NewDefaultEngine's own transitioner parameter
// already establishes in this codebase: this service only ever calls
// Dial.
type dialer interface {
	Dial(ctx context.Context, oltID uuid.UUID) (ssh.Shell, error)
}

// KontronService runs Kontron/Iskratel C16 commands against a specific
// OLT, end to end.
type KontronService struct {
	dial dialer
}

// NewKontronService builds a KontronService.
func NewKontronService(dial dialer) *KontronService {
	return &KontronService{dial: dial}
}

// run opens a connection to oltID, executes fn against it, and always
// closes the connection before returning — regardless of whether fn (or
// the dial itself) succeeded — the one place this service's "always
// close" contract is implemented, so every exported method below is a
// one-line call into it. This matters concretely for a device like the
// C16: leaving a connection open on any exit path, error or not, would
// leak one of its own small number of concurrent session slots (see
// internal/olt/connect's dialedShell doc comment).
//
// Errors are reclassified before being returned: a failure already
// carrying an apperror.Kind (a not-found repository lookup, a Conflict
// from connect.Dialer for missing configuration, an Invalid from
// kontron.ErrInvalidInterface) is returned as-is, but connect.Shell's
// and internal/platform/ssh's own errors (a dial failure, an
// authentication rejection, a command timeout) carry no apperror.Kind of
// their own — internal/platform/ssh deliberately has no dependency on
// apperror at all (see that package's own doc comment on why). Those are
// classified here as apperror.KindUnavailable: "a dependency could not
// be reached" is exactly what a failure to talk to the OLT itself means,
// and httpx.WriteError already hides KindUnavailable's wrapped detail
// from the client (see that function's own doc comment), so nothing
// about the raw SSH failure leaks through the API.
func (s *KontronService) run(ctx context.Context, oltID uuid.UUID, fn func(context.Context, *kontron.Client) (string, error)) (string, error) {
	shell, err := s.dial.Dial(ctx, oltID)
	if err != nil {
		return "", classify("could not reach OLT", err)
	}
	defer func() { _ = shell.Close() }()

	out, err := fn(ctx, kontron.NewClient(shell))
	if err != nil {
		return "", classify("command failed", err)
	}
	return out, nil
}

// classify returns err unchanged if it already carries an apperror.Kind,
// and otherwise wraps it as apperror.KindUnavailable with message — see
// run's own doc comment for the full reasoning. kontron.ErrInvalidInterface
// is the one exception, reclassified as apperror.KindInvalid instead:
// unlike a connectivity failure, an interface value with an embedded
// newline is a caller input problem, not a "device could not be
// reached" one.
func classify(message string, err error) error {
	if errors.Is(err, kontron.ErrInvalidInterface) {
		// apperror.Wrap, not apperror.Invalid(err.Error()): the latter
		// would build a fresh *apperror.Error with no wrapped cause,
		// breaking errors.Is(result, kontron.ErrInvalidInterface) for
		// any caller further up that needs to check specifically for
		// this condition rather than just its apperror.Kind.
		return apperror.Wrap(apperror.KindInvalid, err.Error(), err)
	}

	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return err
	}

	return apperror.Unavailable(message, err)
}

// ONUSummary runs kontron.Client.ONUSummary against the OLT identified
// by oltID.
func (s *KontronService) ONUSummary(ctx context.Context, oltID uuid.UUID) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.ONUSummary(ctx)
	})
}

// ONUStatusSummary runs kontron.Client.ONUStatusSummary against the OLT
// identified by oltID.
func (s *KontronService) ONUStatusSummary(ctx context.Context, oltID uuid.UUID) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.ONUStatusSummary(ctx)
	})
}

// ONURunningConfig runs kontron.Client.ONURunningConfig against the OLT
// identified by oltID, for the ONU on iface.
func (s *KontronService) ONURunningConfig(ctx context.Context, oltID uuid.UUID, iface string) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.ONURunningConfig(ctx, iface)
	})
}

// ONUDetail runs kontron.Client.ONUDetail against the OLT identified by
// oltID, for the ONU on iface.
func (s *KontronService) ONUDetail(ctx context.Context, oltID uuid.UUID, iface string) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.ONUDetail(ctx, iface)
	})
}

// ONUStatus runs kontron.Client.ONUStatus against the OLT identified by
// oltID, for the ONU on iface.
func (s *KontronService) ONUStatus(ctx context.Context, oltID uuid.UUID, iface string) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.ONUStatus(ctx, iface)
	})
}

// ONUEthernetPorts runs kontron.Client.ONUEthernetPorts against the OLT
// identified by oltID, for the ONU on iface.
func (s *KontronService) ONUEthernetPorts(ctx context.Context, oltID uuid.UUID, iface string) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.ONUEthernetPorts(ctx, iface)
	})
}

// DHCPSnoopingEntries runs kontron.Client.DHCPSnoopingEntries against
// the OLT identified by oltID, for the ONU on iface.
func (s *KontronService) DHCPSnoopingEntries(ctx context.Context, oltID uuid.UUID, iface string) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.DHCPSnoopingEntries(ctx, iface)
	})
}

// MACAddressTableEntries runs kontron.Client.MACAddressTableEntries
// against the OLT identified by oltID, for the ONU on iface.
func (s *KontronService) MACAddressTableEntries(ctx context.Context, oltID uuid.UUID, iface string) (string, error) {
	return s.run(ctx, oltID, func(ctx context.Context, c *kontron.Client) (string, error) {
		return c.MACAddressTableEntries(ctx, iface)
	})
}
