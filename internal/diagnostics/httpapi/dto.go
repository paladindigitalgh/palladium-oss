// Package httpapi is the Diagnostics framework's REST layer. It depends
// on internal/diagnostics/service, never on a Registry directly, and
// never exposes internal/diagnostics's domain types over the wire — see
// the DTOs in this file. It mirrors every other domain's httpapi package
// in shape (request DTO in, response DTO out, a thin handler between
// them), even though this one has no repository beneath it at all.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
)

// basicONUCheckRequest is the JSON body for POST
// /api/v1/diagnostics/basic-onu-check.
//
// ONUID is tagged "onuId" — camelCase, not this codebase's otherwise
// universal snake_case ("location_id", "product_id", "access_network_id",
// ...; see e.g. internal/service/httpapi.serviceRequest). This is a
// deliberate, one-off deviation, not a new convention: goal 7 of this
// milestone specifies the exact request body verbatim —
// {"onuId":"..."} — so this field is tagged to match that literal
// instruction exactly. No other field in this codebase should follow
// this casing; a future diagnostics endpoint should return to
// snake_case unless given the same kind of explicit, literal
// instruction this one was.
type basicONUCheckRequest struct {
	ONUID uuid.UUID `json:"onuId"`
}

// toRequest converts a basicONUCheckRequest into a domain
// diagnostics.Request.
func (req basicONUCheckRequest) toRequest() diagnostics.Request {
	return diagnostics.Request{ONUID: req.ONUID}
}

// sectionResponse is the JSON representation of a diagnostics.Section.
type sectionResponse struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Output  string `json:"output"`
}

// resultResponse is the JSON representation of a diagnostics.Result
// returned to clients. Decoupling the wire format from Result's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape — the same
// reasoning every other domain's response DTO in this codebase
// documents.
//
// Duration is rendered via time.Duration.String() (e.g. "1.2ms"), not
// as a raw integer of nanoseconds (Go's default json.Marshal behavior
// for a time.Duration, which is just an int64 underneath): a bare
// nanosecond count is not something an operator reading this response
// can interpret at a glance, and no other numeric-with-implicit-unit
// field exists elsewhere in this codebase to be consistent with instead.
type resultResponse struct {
	Name       string            `json:"name"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt time.Time         `json:"finished_at"`
	Duration   string            `json:"duration"`
	Sections   []sectionResponse `json:"sections"`
}

func newResultResponse(result diagnostics.Result) resultResponse {
	sections := make([]sectionResponse, len(result.Sections))
	for i, s := range result.Sections {
		sections[i] = sectionResponse{Name: s.Name, Command: s.Command, Output: s.Output}
	}

	return resultResponse{
		Name:       result.Name,
		StartedAt:  result.StartedAt,
		FinishedAt: result.FinishedAt,
		Duration:   result.Duration.String(),
		Sections:   sections,
	}
}
