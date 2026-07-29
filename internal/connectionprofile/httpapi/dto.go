// Package httpapi is the Connection Profile domain's REST layer. It
// depends on internal/connectionprofile/service, never on a repository
// directly, and never exposes internal/connectionprofile's domain types
// over the wire — see the DTOs in this file. It mirrors
// internal/catalog/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
)

// connectionProfileRequest is the JSON body for POST
// /api/v1/connection-profiles and PUT /api/v1/connection-profiles/{id}.
//
// HostKeyPolicy is a plain string here, not
// connectionprofile.HostKeyPolicy, even though that type would marshal
// to the same JSON today — the same "DTOs only" separation
// internal/catalog/httpapi.catalogRequest documents. Protocol is
// likewise a plain string, but for a different reason: the domain field
// itself is a plain string, not an enum (see
// internal/connectionprofile/validate.go's own doc comment on why).
//
// Timeout is a string, not a raw number of anything: "30s", parsed via
// time.ParseDuration, the same human-readable representation
// internal/diagnostics/httpapi chose for Result.Duration — a caller
// writes exactly what they mean (seconds, minutes, ...) rather than
// having to know or guess which unit a bare integer is counting.
//
// AuthenticationID is a plain *uuid.UUID: nil (JSON null, or the field
// omitted) means "no Authentication bound to this profile," matching
// connectionprofile.ConnectionProfile.AuthenticationID's own optionality
// (see that package's model.go).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type connectionProfileRequest struct {
	Name             string     `json:"name"`
	Protocol         string     `json:"protocol"`
	Port             int        `json:"port"`
	AuthenticationID *uuid.UUID `json:"authentication_id"`
	Timeout          string     `json:"timeout"`
	HostKeyPolicy    string     `json:"host_key_policy"`
	Description      string     `json:"description"`
}

// toConnectionProfile converts a request into a domain
// connectionprofile.ConnectionProfile. id is supplied by the caller:
// uuid.Nil for Create (the repository assigns a real one), or the URL
// path parameter's UUID for Update. An empty or unparseable Timeout is
// treated as zero (no timeout configured) rather than an error here —
// ConnectionProfile.Validate does not require Timeout at all (see this
// milestone's "only Name unique" reading, documented in
// internal/connectionprofile/validate.go), so a malformed Timeout string
// is not this conversion step's problem to reject; it simply does not
// set one.
func (req connectionProfileRequest) toConnectionProfile(id uuid.UUID) connectionprofile.ConnectionProfile {
	timeout, _ := time.ParseDuration(req.Timeout)

	return connectionprofile.ConnectionProfile{
		ID:               id,
		Name:             req.Name,
		Protocol:         req.Protocol,
		Port:             req.Port,
		AuthenticationID: req.AuthenticationID,
		Timeout:          timeout,
		HostKeyPolicy:    connectionprofile.HostKeyPolicy(req.HostKeyPolicy),
		Description:      req.Description,
	}
}

// connectionProfileResponse is the JSON representation of a
// ConnectionProfile returned to clients. Decoupling the wire format from
// ConnectionProfile's Go field layout and types means a change to how
// the domain model is composed internally can never silently change the
// API's JSON shape.
type connectionProfileResponse struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Protocol         string     `json:"protocol"`
	Port             int        `json:"port"`
	AuthenticationID *uuid.UUID `json:"authentication_id"`
	Timeout          string     `json:"timeout"`
	HostKeyPolicy    string     `json:"host_key_policy"`
	Description      string     `json:"description"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func newConnectionProfileResponse(p connectionprofile.ConnectionProfile) connectionProfileResponse {
	return connectionProfileResponse{
		ID:               p.ID,
		Name:             p.Name,
		Protocol:         p.Protocol,
		Port:             p.Port,
		AuthenticationID: p.AuthenticationID,
		Timeout:          p.Timeout.String(),
		HostKeyPolicy:    string(p.HostKeyPolicy),
		Description:      p.Description,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// connectionProfileListResponse wraps a slice of profiles in an object
// rather than returning a bare JSON array — the same reasoning as
// internal/catalog/httpapi's catalogListResponse: a bare top-level array
// can never gain sibling fields (a total count, a pagination cursor,
// ...) without becoming a breaking change for existing clients, while
// adding a field next to "connection_profiles" is not.
type connectionProfileListResponse struct {
	ConnectionProfiles []connectionProfileResponse `json:"connection_profiles"`
}

func newConnectionProfileListResponse(profiles []connectionprofile.ConnectionProfile) connectionProfileListResponse {
	resp := connectionProfileListResponse{ConnectionProfiles: make([]connectionProfileResponse, len(profiles))}
	for i, p := range profiles {
		resp.ConnectionProfiles[i] = newConnectionProfileResponse(p)
	}
	return resp
}
