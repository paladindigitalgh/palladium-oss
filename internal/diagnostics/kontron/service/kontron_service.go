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
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
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

// oltLister is the seam AggregatedBlacklist uses to find every OLT
// worth checking, narrowed to the one method it calls — the same
// consumer-defined-interface pattern dialer above already establishes
// in this file.
type oltLister interface {
	List(ctx context.Context) ([]olt.OLT, error)
}

// oltModelGetter is the seam AggregatedBlacklist uses to learn an OLT's
// Vendor (an OLT does not carry Vendor itself — see internal/oltmodel's
// own doc comment on why that lives on OLTModel), narrowed the same way
// oltLister is.
type oltModelGetter interface {
	Get(ctx context.Context, id uuid.UUID) (oltmodel.OLTModel, error)
}

// KontronService runs Kontron/Iskratel C16 commands against a specific
// OLT, end to end.
//
// It composes three things, not just dial: olts and oltModels exist
// solely for AggregatedBlacklist, which — unlike every other method
// here — is not scoped to one already-known OLT. It has to discover
// which OLTs even exist and which of those are Kontron equipment before
// it can dial any of them. This mirrors two precedents already
// established elsewhere in this codebase for the same reason (a real
// cross-domain effect with no narrower place to live): internal/olt/
// service.OLTService composes olt+oltmodel+ponport for its own
// auto-port-creation cascade, and internal/workflow/engine.DefaultEngine
// composes service+serviceequipment+a plugin registry for its own
// execution. Every other KontronService method below still depends on
// dial alone.
type KontronService struct {
	dial      dialer
	olts      oltLister
	oltModels oltModelGetter
}

// NewKontronService builds a KontronService.
func NewKontronService(dial dialer, olts oltLister, oltModels oltModelGetter) *KontronService {
	return &KontronService{dial: dial, olts: olts, oltModels: oltModels}
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

// BlacklistedONU is one AggregatedBlacklist row: a kontron.BlacklistEntry
// tagged with which OLT it came from, since a caller aggregating across
// every Kontron OLT on the network has no other way to tell them apart.
type BlacklistedONU struct {
	OLTID          uuid.UUID
	OLTName        string
	Interface      string
	SerialNumber   string
	RegistrationID string
	Cause          string
}

// UnreachableOLT names a Kontron OLT AggregatedBlacklist could not get a
// usable answer from — either the OLT itself (dial/command failure) or
// its response (a kontron.ParseBlacklistEntries failure) — and why.
type UnreachableOLT struct {
	OLTID   uuid.UUID
	OLTName string
	Reason  string
}

// AggregatedBlacklist is AggregatedBlacklist's result: every blacklisted
// ONU found across every reachable Kontron OLT, plus which OLTs (if any)
// could not be checked.
type AggregatedBlacklist struct {
	ONUs        []BlacklistedONU
	Unreachable []UnreachableOLT
}

// AggregatedBlacklist runs kontron.Client.BlacklistedONUs against every
// Kontron-vendor OLT on the network, concurrently, and merges the
// results into one list.
//
// This is triggered on demand by a caller (an operator about to bring a
// new ONU into service, confirming its serial number and that it is
// actually online before typing it in by hand) — nothing here persists,
// schedules, or caches this result anywhere; every call re-queries every
// OLT from scratch, by design.
//
// Non-Kontron OLTs are filtered out before dialing anything: this
// service only knows how to speak to a Kontron C16 (see this package's
// own doc comment), so sending BlacklistedONUs' command to an OLT of an
// unknown or different vendor is not attempted at all, rather than
// dialed and hoped to fail cleanly. A failure to load a given OLT's
// OLTModel (the lookup itself erroring, not merely reporting a
// non-Kontron vendor) is treated as a real data-integrity problem — the
// foreign key from olts.olt_model_id already guarantees a valid OLTModel
// row exists — and fails this call entirely rather than being folded
// into Unreachable, the same distinction
// internal/olt/service.OLTService.Create's own doc comment draws between
// "a dependency it needs is broken" and "this one instance could not be
// reached."
//
// Every reachable Kontron OLT is dialed concurrently, not one at a time:
// this call's whole reason to exist is answering "what is out there
// right now" fast enough to use interactively, and this codebase's
// expected OLT fleet size does not call for a bounded worker pool to get
// there.
func (s *KontronService) AggregatedBlacklist(ctx context.Context) (AggregatedBlacklist, error) {
	allOLTs, err := s.olts.List(ctx)
	if err != nil {
		return AggregatedBlacklist{}, err
	}

	var kontronOLTs []olt.OLT
	for _, o := range allOLTs {
		model, err := s.oltModels.Get(ctx, o.OLTModelID)
		if err != nil {
			return AggregatedBlacklist{}, fmt.Errorf("kontron: load olt model for olt %s: %w", o.ID, err)
		}
		if model.Vendor == oltmodel.VendorKontron {
			kontronOLTs = append(kontronOLTs, o)
		}
	}

	type outcome struct {
		onus        []BlacklistedONU
		unreachable *UnreachableOLT
	}
	outcomes := make([]outcome, len(kontronOLTs))

	var wg sync.WaitGroup
	for i, o := range kontronOLTs {
		wg.Add(1)
		go func(i int, o olt.OLT) {
			defer wg.Done()

			raw, err := s.run(ctx, o.ID, func(ctx context.Context, c *kontron.Client) (string, error) {
				return c.BlacklistedONUs(ctx)
			})
			if err != nil {
				outcomes[i] = outcome{unreachable: &UnreachableOLT{OLTID: o.ID, OLTName: o.Name, Reason: err.Error()}}
				return
			}

			entries, err := kontron.ParseBlacklistEntries(raw)
			if err != nil {
				outcomes[i] = outcome{unreachable: &UnreachableOLT{OLTID: o.ID, OLTName: o.Name, Reason: "unexpected response format: " + err.Error()}}
				return
			}

			onus := make([]BlacklistedONU, len(entries))
			for j, e := range entries {
				onus[j] = BlacklistedONU{
					OLTID:          o.ID,
					OLTName:        o.Name,
					Interface:      e.Interface,
					SerialNumber:   e.SerialNumber,
					RegistrationID: e.RegistrationID,
					Cause:          e.Cause,
				}
			}
			outcomes[i] = outcome{onus: onus}
		}(i, o)
	}
	wg.Wait()

	var result AggregatedBlacklist
	for _, o := range outcomes {
		if o.unreachable != nil {
			result.Unreachable = append(result.Unreachable, *o.unreachable)
			continue
		}
		result.ONUs = append(result.ONUs, o.onus...)
	}
	return result, nil
}
