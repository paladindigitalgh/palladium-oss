package diagnostics

import (
	"context"
	"time"
)

// BasicONUCheckName is BasicONUCheck's registration name (see Registry)
// and the value its Result.Name carries. Exported so
// internal/diagnostics/httpapi can name this specific, known diagnostic
// when handling POST /api/v1/diagnostics/basic-onu-check — that route is
// fixed to this one diagnostic, not a generic "run whatever name is in
// the URL" endpoint (this milestone defines exactly one built-in
// diagnostic and exactly one endpoint; a future milestone adding more of
// either is free to introduce that generality when it actually has a
// second case to generalize over).
const BasicONUCheckName = "Basic ONU Check"

// BasicONUCheck is this milestone's one built-in Diagnostic: a
// placeholder that proves the framework itself — Registry registration,
// DiagnosticsService lookup and execution, HTTP request/response — works
// end to end, without yet touching any network device. Per this
// milestone's explicit scope, it does not open an SSH session, does not
// run a CLI command, and does not parse any command output; Run always
// succeeds and always returns the same fixed Result.
//
// BasicONUCheck holds no fields and NewBasicONUCheck takes no
// arguments — there is nothing to configure yet. A future real
// diagnostic (one that actually reaches an ONU) would plausibly need
// something like an SSH client or a topology resolver injected here;
// this placeholder needs none of that, so it has none of that, per
// CLAUDE.md's general rule to prefer the simpler option and avoid
// designing for hypothetical future requirements.
type BasicONUCheck struct{}

// NewBasicONUCheck builds a BasicONUCheck.
func NewBasicONUCheck() BasicONUCheck {
	return BasicONUCheck{}
}

var _ Diagnostic = BasicONUCheck{}

// Name implements Diagnostic.
func (BasicONUCheck) Name() string {
	return BasicONUCheckName
}

// Run implements Diagnostic. It ignores request entirely (there is
// nothing yet to do with an ONUID — see Request's doc comment on why
// topology resolution is deliberately not this milestone's job) and
// always returns a successful Result with exactly one Section, per
// goal 5: Command and Output are both the literal string "not
// implemented," making unmistakably clear to any caller that this is a
// framework placeholder, not a real diagnostic result that happens to
// have come back empty.
func (BasicONUCheck) Run(_ context.Context, _ Request) (*Result, error) {
	startedAt := time.Now()
	finishedAt := time.Now()

	return &Result{
		Name:       BasicONUCheckName,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Duration:   finishedAt.Sub(startedAt),
		Sections: []Section{
			{
				Name:    BasicONUCheckName,
				Command: "not implemented",
				Output:  "not implemented",
			},
		},
	}, nil
}
