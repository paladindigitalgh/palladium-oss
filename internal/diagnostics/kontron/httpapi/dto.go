// Package httpapi is the Kontron diagnostics REST layer. It depends on
// internal/diagnostics/kontron/service, never on
// internal/diagnostics/kontron or internal/olt/connect directly, and
// never exposes their types over the wire — see the DTOs in this file.
// It mirrors every other domain's httpapi package in shape, even though
// this one has no repository beneath it at all.
package httpapi

import (
	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron/service"
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

// blacklistedONUResponse is one onu-blacklist response row — the wire
// shape of service.BlacklistedONU. Unlike commandOutputResponse, this
// one is structured, not raw text: see
// internal/diagnostics/kontron/blacklist.go's own doc comment on why
// this one command's output is parsed at all.
type blacklistedONUResponse struct {
	OLTID          string `json:"olt_id"`
	OLTName        string `json:"olt_name"`
	Interface      string `json:"interface"`
	SerialNumber   string `json:"serial_number"`
	RegistrationID string `json:"registration_id"`
	Cause          string `json:"cause"`
}

// unreachableOLTResponse is the wire shape of service.UnreachableOLT.
type unreachableOLTResponse struct {
	OLTID   string `json:"olt_id"`
	OLTName string `json:"olt_name"`
	Reason  string `json:"reason"`
}

// onuBlacklistResponse is the JSON representation of
// service.AggregatedBlacklist.
type onuBlacklistResponse struct {
	ONUs            []blacklistedONUResponse `json:"onus"`
	UnreachableOLTs []unreachableOLTResponse `json:"unreachable_olts"`
}

// newONUBlacklistResponse converts a service.AggregatedBlacklist to its
// wire shape. ONUs and UnreachableOLTs are built as non-nil empty slices
// even when agg's own fields are nil, so a fully-reachable, fully-empty
// network reports "[]" rather than "null" for either — the same
// "empty means empty, not absent" convention every other list response
// in this codebase's httpapi packages already follows.
func newONUBlacklistResponse(agg service.AggregatedBlacklist) onuBlacklistResponse {
	onus := make([]blacklistedONUResponse, len(agg.ONUs))
	for i, o := range agg.ONUs {
		onus[i] = blacklistedONUResponse{
			OLTID:          o.OLTID.String(),
			OLTName:        o.OLTName,
			Interface:      o.Interface,
			SerialNumber:   o.SerialNumber,
			RegistrationID: o.RegistrationID,
			Cause:          o.Cause,
		}
	}

	unreachable := make([]unreachableOLTResponse, len(agg.Unreachable))
	for i, u := range agg.Unreachable {
		unreachable[i] = unreachableOLTResponse{
			OLTID:   u.OLTID.String(),
			OLTName: u.OLTName,
			Reason:  u.Reason,
		}
	}

	return onuBlacklistResponse{ONUs: onus, UnreachableOLTs: unreachable}
}
