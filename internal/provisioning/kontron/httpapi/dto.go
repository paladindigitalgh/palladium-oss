// Package httpapi is the Kontron ONU-authorization REST layer. It
// depends on internal/provisioning/kontron/service, never on
// internal/provisioning/kontron or internal/olt/connect directly, and
// never exposes their types over the wire — see the DTOs in this file.
// It mirrors internal/diagnostics/kontron/httpapi's shape exactly, even
// though this one has no repository beneath it at all.
package httpapi

import "github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"

// authorizeONURequest is the JSON body for POST
// /api/v1/provisioning/olts/{oltId}/authorize-onu.
//
// Neither field is validated for shape beyond non-empty here: Port comes
// from Palladium's own blacklist-check data (see
// internal/diagnostics/kontron.BlacklistEntry.Interface) once a caller
// has it, and SerialNumber likewise — see
// internal/provisioning/kontron.Client's own ErrInvalidInterface/
// ErrInvalidSerialNumber doc comments for the one thing that actually is
// validated (no embedded newline), enforced further down the stack, not
// here.
type authorizeONURequest struct {
	Port         string `json:"port"`
	SerialNumber string `json:"serial_number"`
}

func (req authorizeONURequest) validate() error {
	if req.Port == "" {
		return apperror.Invalid("port is required")
	}
	if req.SerialNumber == "" {
		return apperror.Invalid("serial_number is required")
	}
	return nil
}

// authorizeONUResponse is the JSON representation of AuthorizeONU's
// result: the interface it assigned the new ONU. DeauthorizationHandler
// reuses this same shape for DeauthorizeONU's result — both are exactly
// "the interface this call acted on," so a second, identically-shaped
// type would add nothing.
type authorizeONUResponse struct {
	Interface string `json:"interface"`
}
