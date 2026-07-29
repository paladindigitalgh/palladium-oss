// Package httpapi is the Authentication domain's REST layer. It depends
// on internal/authentication/service, never on a repository directly,
// and never exposes internal/authentication's domain types over the
// wire — see the DTOs in this file. It mirrors internal/catalog/httpapi
// in shape, with one deliberate departure explained below.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
)

// authenticationRequest is the JSON body for POST
// /api/v1/authentication-methods and PUT
// /api/v1/authentication-methods/{id}.
//
// AuthenticationType is a plain string here, not
// authentication.AuthenticationType, even though that type would marshal
// to the same JSON today — the same "DTOs only" separation
// internal/catalog/httpapi.catalogRequest documents. The conversion
// happens once, explicitly, in toAuthentication below;
// AuthenticationService.Create/Update reject an unrecognized value via
// Authentication.Validate (see internal/authentication/validate.go)
// exactly as they would for a request built any other way — this
// handler does not duplicate that check (the service is where
// validation lives).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type authenticationRequest struct {
	Name               string `json:"name"`
	AuthenticationType string `json:"authentication_type"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	PrivateKey         string `json:"private_key"`
}

// toAuthentication converts a request into a domain
// authentication.Authentication. id is supplied by the caller: uuid.Nil
// for Create (the repository assigns a real one), or the URL path
// parameter's UUID for Update.
func (req authenticationRequest) toAuthentication(id uuid.UUID) authentication.Authentication {
	return authentication.Authentication{
		ID:                 id,
		Name:               req.Name,
		AuthenticationType: authentication.AuthenticationType(req.AuthenticationType),
		Username:           req.Username,
		Password:           req.Password,
		PrivateKey:         req.PrivateKey,
	}
}

// authenticationResponse is the JSON representation of an Authentication
// returned to clients.
//
// It deliberately omits Password and PrivateKey entirely, even though
// authentication.Authentication holds their real, decrypted values in
// memory (see that package's doc comment) — the one departure from
// every other domain's response DTO in this codebase, which otherwise
// echo back everything they were given. Returning a secret's plaintext
// over a read API is a real exposure this milestone's "Secrets must
// never be stored in plaintext" principle argues against in spirit even
// though it is worded about storage specifically: an operator's browser
// history, a proxy's access log, or a misdirected response would all
// then carry the plaintext credential. HasPassword and HasPrivateKey
// exist instead, so a caller (and per goal 7, a future UI) can render
// "Password: configured" versus "not set" without the value ever
// reaching the wire — a create or update response looks identical to a
// get/list response for exactly the same reason: even immediately
// echoing back what was just submitted is an avoidable exposure a
// sensible UI does not need.
type authenticationResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	AuthenticationType string    `json:"authentication_type"`
	Username           string    `json:"username"`
	HasPassword        bool      `json:"has_password"`
	HasPrivateKey      bool      `json:"has_private_key"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func newAuthenticationResponse(a authentication.Authentication) authenticationResponse {
	return authenticationResponse{
		ID:                 a.ID,
		Name:               a.Name,
		AuthenticationType: string(a.AuthenticationType),
		Username:           a.Username,
		HasPassword:        a.Password != "",
		HasPrivateKey:      a.PrivateKey != "",
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
	}
}

// authenticationListResponse wraps a slice of authentications in an
// object rather than returning a bare JSON array — the same reasoning
// as internal/catalog/httpapi's catalogListResponse: a bare top-level
// array can never gain sibling fields (a total count, a pagination
// cursor, ...) without becoming a breaking change for existing clients,
// while adding a field next to "authentication_methods" is not.
type authenticationListResponse struct {
	AuthenticationMethods []authenticationResponse `json:"authentication_methods"`
}

func newAuthenticationListResponse(auths []authentication.Authentication) authenticationListResponse {
	resp := authenticationListResponse{AuthenticationMethods: make([]authenticationResponse, len(auths))}
	for i, a := range auths {
		resp.AuthenticationMethods[i] = newAuthenticationResponse(a)
	}
	return resp
}
