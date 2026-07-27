// Package httpx holds small, domain-agnostic helpers shared by every HTTP
// handler in the project: writing a JSON response, decoding a JSON
// request body, and translating an internal/platform/apperror into an
// HTTP status code and response body.
//
// This exists because goal 5 ("translate platform errors into HTTP
// responses") is not a Site-specific concern — every future entity's
// handlers (Building, Room, Rack, Device, and beyond) need the exact same
// apperror.Kind -> status mapping. Putting it here once, rather than
// inside internal/inventory/httpapi, is what keeps it from being
// reimplemented (and potentially drifting) in every domain's handler
// package. It depends only on net/http, encoding/json, and
// internal/platform/apperror — no domain knowledge at all.
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// WriteJSON writes body as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// DecodeJSON decodes r's body into dst. Unknown fields are rejected: DTOs
// (internal/inventory/httpapi's request types, and every future domain's)
// define an explicit contract, and silently ignoring a field the client
// thought it was setting — most often a typo — is worse than telling them
// immediately. Any decode failure is returned as an apperror.KindInvalid
// error, ready to pass straight to WriteError.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperror.Invalid("request body is not valid JSON")
	}
	return nil
}

// errorResponse is the JSON shape of every error this package writes.
// Kind is included, not just a human-readable message: it gives API
// clients something stable to branch on programmatically without parsing
// the message text or relying solely on the HTTP status code (several
// Kinds could plausibly share a status in the future).
type errorResponse struct {
	Error string `json:"error"`
	Kind  string `json:"kind"`
}

// WriteError translates err into an HTTP status code and JSON body via
// apperror.KindOf, and writes it.
//
// KindInternal and KindUnavailable never have their message shown to the
// client, even though *apperror.Error.Message is normally safe to expose
// for the other kinds. This is deliberate, not an oversight: those two
// kinds are exactly the ones a repository's error-translation layer
// attaches a wrapped underlying cause to (see e.g.
// internal/inventory/postgres/errors.go's apperror.Internal calls, which
// wrap the raw pgx/PostgreSQL error). *apperror.Error.Error() includes
// that wrapped cause in its output, so calling it here would risk leaking
// exactly what goal 5 says to keep hidden: "keep PostgreSQL errors hidden
// from clients." Using a fixed, generic message for those two kinds makes
// that guarantee structural — true regardless of what any particular
// call site happened to pass in — rather than something that only holds
// as long as every caller remembers to be careful.
func WriteError(w http.ResponseWriter, err error) {
	kind := apperror.KindOf(err)

	status, message := http.StatusInternalServerError, "an internal error occurred"
	switch kind {
	case apperror.KindInvalid:
		status, message = http.StatusBadRequest, safeMessage(err)
	case apperror.KindNotFound:
		status, message = http.StatusNotFound, safeMessage(err)
	case apperror.KindConflict:
		status, message = http.StatusConflict, safeMessage(err)
	case apperror.KindUnauthorized:
		status, message = http.StatusUnauthorized, safeMessage(err)
	case apperror.KindForbidden:
		status, message = http.StatusForbidden, safeMessage(err)
	case apperror.KindUnavailable:
		status, message = http.StatusServiceUnavailable, "service temporarily unavailable"
	}
	// KindInternal, and anything not in the switch above, keeps the
	// http.StatusInternalServerError / "an internal error occurred"
	// default set before the switch.

	WriteJSON(w, status, errorResponse{Error: message, Kind: string(kind)})
}

// safeMessage returns err's message without ever including a wrapped
// cause. *apperror.Error.Error() appends ": <cause>" whenever Err is set
// (see internal/platform/apperror/apperror.go); reading Message directly
// instead sidesteps that. Only called for the kinds WriteError treats as
// safe to describe to the client (see its doc comment) — never for
// KindInternal or KindUnavailable.
func safeMessage(err error) string {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return "invalid request"
}
