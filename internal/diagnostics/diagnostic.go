// Package diagnostics defines the framework future vendor-specific
// diagnostics will plug into — the Diagnostic interface, the Request and
// Result shapes every diagnostic speaks, and a Registry diagnostics
// register themselves into by name. Per this milestone's explicit scope,
// it connects to no network device: no SSH, no CLI execution, no command
// parsing, no vendor logic. Those belong to concrete diagnostics a future
// milestone writes against this framework — see
// internal/provisioning/connectors for the same "define the interface
// now, implement vendor-specific behavior later" shape applied to
// provisioning (CLAUDE.md's Plugin Philosophy: "Everything vendor-
// specific belongs in plugins. The core system must never contain
// vendor-specific logic.").
//
// This package has no dependency beyond the standard library and
// github.com/google/uuid (already pervasive throughout this codebase) —
// no database driver, no HTTP framework, no vendor SDK. A Diagnostic
// implementation is free to add its own dependencies (an SSH client
// library, say) without this package ever needing to know about them.
package diagnostics

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Request is what every Diagnostic's Run receives: which ONU to
// diagnose, identified the same way every other domain in this codebase
// identifies a record — a uuid.UUID — and nothing else. Per this
// milestone's explicit instruction, Request does not carry a resolved
// AccessAttachment, AccessInterface, OLT, or any other piece of
// topology: "the framework will resolve topology internally in future
// milestones," so a Diagnostic implementation asks for an ONU by ID and
// trusts the framework (not yet built) to have worked out what that
// means operationally by the time Run is called.
type Request struct {
	ONUID uuid.UUID
}

// Section is one named piece of a Result — conceptually, "the output of
// one command a diagnostic ran." Command and Output are both plain
// strings, not structured data: this milestone does not implement
// command parsing (see this package's doc comment), so a Section's
// Output is whatever raw text the underlying operation produced,
// verbatim, with no attempt to interpret it. A future diagnostic backed
// by a real SSH session might have several Sections, one per command it
// ran; the placeholder built into this milestone (see
// basic_onu_check.go) has exactly one.
type Section struct {
	Name    string
	Command string
	Output  string
}

// Result is what every Diagnostic's Run returns on success: which
// diagnostic ran, when it started and finished, how long it took, and
// the Sections of output it produced. Duration is stored explicitly
// rather than always being computed as FinishedAt.Sub(StartedAt) at the
// point of use, the same reasoning a stored, derived field earns
// anywhere in this codebase when multiple callers (here: the HTTP layer
// formatting a response) would otherwise all need to recompute it
// identically.
//
// Result carries no persistence-related fields (no ID, no CreatedAt/
// UpdatedAt) because none exist: per this milestone's explicit scope,
// there is no repository, no database write, nothing to assign an
// identity to. A Result lives exactly as long as the request that asked
// for it.
type Result struct {
	Name       string
	StartedAt  time.Time
	FinishedAt time.Time
	Duration   time.Duration
	Sections   []Section
}

// Diagnostic is the abstraction every future vendor-specific diagnostic
// will implement — an SSH-backed ONU signal check, a Kontron-specific
// port status query, and so on (see this package's doc comment for why
// none of those exist yet). It mirrors
// internal/provisioning/connectors.Connector's shape deliberately: one
// Name method for registration and lookup, one operation method that
// takes a context and a request-shaped argument and returns a result (or
// an error).
//
// Run returning (*Result, error) rather than (Result, error) mirrors
// this codebase's established convention for "this operation might
// legitimately produce nothing" (see e.g. every domain's Get returning a
// zero value on error): a failed diagnostic run has no meaningful
// partial Result to hand back, so returning nil alongside a non-nil
// error is unambiguous, whereas a zero-value Result could be mistaken
// for a real (if empty) one.
type Diagnostic interface {
	// Name identifies this Diagnostic for registration and lookup (see
	// Registry) and is the value stored in the Result it produces.
	Name() string

	Run(ctx context.Context, request Request) (*Result, error)
}
