// Package service is the Kontron ONU-authorization caller: it ties
// internal/olt/connect.Dialer (resolve an OLT by ID into a live shell)
// together with internal/diagnostics/kontron.Client (read the OLT's
// current ONU assignments) and internal/provisioning/kontron.Client (run
// the authorize-and-save command sequence), always closing the
// connection afterward regardless of outcome. It mirrors
// internal/diagnostics/kontron/service's exact shape — including
// duplicating its dialer interface and its dial/defer-close/classify
// pattern rather than sharing it — because that pattern is unexported
// there; a small amount of duplication is the accepted cost of this
// package boundary (see internal/provisioning/kontron's own doc comment
// on why the read-only and write-capable Kontron packages must not
// depend on each other in the wrong direction).
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	diagnosticskontron "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
	provisioningkontron "github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron"
)

// dialer is the seam AuthorizationService depends on instead of the
// concrete *connect.Dialer — the same narrowing pattern
// internal/diagnostics/kontron/service's own dialer interface already
// establishes in this codebase.
type dialer interface {
	Dial(ctx context.Context, oltID uuid.UUID) (ssh.Shell, error)
}

// AuthorizationService runs the Kontron ONU-authorization sequence
// against a specific OLT, end to end.
type AuthorizationService struct {
	dial                     dialer
	managementServiceProfile string
}

// NewAuthorizationService builds an AuthorizationService. managementServiceProfile
// names the service profile applied to every ONU this service authorizes
// (see provisioningkontron.Client.AuthorizeONU's own doc comment on why
// it is a caller-supplied value, not a literal in that package) —
// internal/config.KontronConfig.ManagementServiceProfile is its one real
// caller, defaulting to "iphost".
func NewAuthorizationService(dial dialer, managementServiceProfile string) *AuthorizationService {
	return &AuthorizationService{dial: dial, managementServiceProfile: managementServiceProfile}
}

// AuthorizeONU opens one connection to oltID, reads its current ONU
// assignments to compute the next free index on port (see
// provisioningkontron.NextFreeIndex), then runs the full
// authorize-and-save sequence for serialNumber on the resulting
// interface — one SSH session for the whole operation, so nothing else
// can claim the same index in between the read and the write. Returns
// the interface it assigned (e.g. "xgs/6/7") on success.
//
// Both a diagnosticskontron.Client (for the read) and a
// provisioningkontron.Client (for the write) are built over the same
// shell: this is one continuous operation against one device, not two
// independent calls that happen to run back to back.
//
// Errors are reclassified the same way
// internal/diagnostics/kontron/service's own classify does, for the
// identical reasoning given there: a failure already carrying an
// apperror.Kind is returned as-is; provisioningkontron's
// ErrInvalidInterface/ErrInvalidSerialNumber become apperror.KindInvalid
// (a caller input problem); everything else from the SSH/dial layer
// (which carries no apperror.Kind of its own — see
// internal/platform/ssh's own doc comment on why) becomes
// apperror.KindUnavailable.
func (s *AuthorizationService) AuthorizeONU(ctx context.Context, oltID uuid.UUID, port, serialNumber string) (string, error) {
	shell, err := s.dial.Dial(ctx, oltID)
	if err != nil {
		return "", classify("could not reach OLT", err)
	}
	defer func() { _ = shell.Close() }()

	summary, err := diagnosticskontron.NewClient(shell).ONUSummary(ctx)
	if err != nil {
		return "", classify("could not read current ONU assignments", err)
	}

	entries, err := diagnosticskontron.ParseONUSummaryEntries(summary)
	if err != nil {
		return "", apperror.Unavailable("could not parse current ONU assignments", err)
	}

	index := provisioningkontron.NextFreeIndex(entries, port)
	iface := fmt.Sprintf("%s/%d", port, index)

	if err := provisioningkontron.NewClient(shell).AuthorizeONU(ctx, iface, serialNumber, s.managementServiceProfile); err != nil {
		return "", classify("command failed", err)
	}

	return iface, nil
}

// classify returns err unchanged if it already carries an apperror.Kind,
// and otherwise wraps it as apperror.KindUnavailable with message — see
// AuthorizeONU's own doc comment for the full reasoning, and
// internal/diagnostics/kontron/service's own classify for the sibling
// this mirrors. provisioningkontron.ErrInvalidInterface and
// ErrInvalidSerialNumber are the two exceptions, reclassified as
// apperror.KindInvalid instead: unlike a connectivity failure, an
// interface, serial number, or management service profile value with an
// embedded newline is a caller input problem, not a "device could not be
// reached" one.
func classify(message string, err error) error {
	if errors.Is(err, provisioningkontron.ErrInvalidInterface) ||
		errors.Is(err, provisioningkontron.ErrInvalidSerialNumber) ||
		errors.Is(err, provisioningkontron.ErrInvalidManagementServiceProfile) {
		// apperror.Wrap, not apperror.Invalid(err.Error()): the latter
		// would build a fresh *apperror.Error with no wrapped cause,
		// breaking errors.Is(result, provisioningkontron.ErrInvalid...)
		// for any caller further up that needs to check specifically for
		// this condition rather than just its apperror.Kind.
		return apperror.Wrap(apperror.KindInvalid, err.Error(), err)
	}

	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return err
	}

	return apperror.Unavailable(message, err)
}
