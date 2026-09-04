// Package httpapi is the Kontron diagnostics REST layer. It depends on
// internal/diagnostics/kontron/service, never on
// internal/diagnostics/kontron or internal/olt/connect directly, and
// never exposes their types over the wire — see the DTOs in this file.
// It mirrors every other domain's httpapi package in shape, even though
// this one has no repository beneath it at all.
package httpapi

import (
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// interfaceRequest is the JSON body for every per-interface endpoint
// below (onu-running-config, onu-detail, onu-status, onu-ethernet-ports,
// dhcp-snooping-entries, mac-address-table-entries).
//
// Interface is not validated for shape here (compare to, say, a
// well-formed hostname or IP address check) — see
// internal/diagnostics/kontron's own ErrInvalidInterface doc comment for
// why: it is expected to come from Palladium's own stored data (an OLT
// or ONU record) once that exists, not typed into a form, so there is no
// format to validate against yet. It is still required to be non-empty
// (see validate), and kontron.Client itself still guards against an
// embedded newline regardless of what called it — this DTO's own
// validate is a cheap, obvious rejection for the one case any caller,
// trusted or not, would never intend: nothing at all.
type interfaceRequest struct {
	Interface string `json:"interface"`
}

func (req interfaceRequest) validate() error {
	if req.Interface == "" {
		return apperror.Invalid("interface is required")
	}
	return nil
}

// commandOutputResponse is the JSON representation of a Kontron command's
// result: the device's raw output, verbatim, exactly as
// internal/diagnostics/kontron's own "no parsing" guarantee promises —
// there is no structured field-by-field breakdown to return because none
// exists yet.
type commandOutputResponse struct {
	Output string `json:"output"`
}
