package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// authenticationService is the seam AuthenticationHandler depends on
// instead of a concrete *service.AuthenticationService — the same
// reasoning internal/catalog/httpapi's catalogService interface
// documents: it lets handler tests exercise HTTP behavior (status codes,
// JSON shapes, routing, error mapping) against a fake, with no real
// service, repository, or database involved. Unexported for the same
// reason catalogService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type authenticationService interface {
	Get(ctx context.Context, id uuid.UUID) (authentication.Authentication, error)
	List(ctx context.Context) ([]authentication.Authentication, error)
	Create(ctx context.Context, a authentication.Authentication) (authentication.Authentication, error)
	Update(ctx context.Context, a authentication.Authentication) (authentication.Authentication, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// AuthenticationHandler serves the Authentication REST endpoints:
//
//	POST   /api/v1/authentication-methods
//	GET    /api/v1/authentication-methods
//	GET    /api/v1/authentication-methods/{id}
//	PUT    /api/v1/authentication-methods/{id}
//	DELETE /api/v1/authentication-methods/{id}
//
// It depends only on authenticationService — never a repository directly
// — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is AuthenticationService's job. See dto.go's own
// doc comment for why this handler's responses never include Password
// or PrivateKey.
type AuthenticationHandler struct {
	authentications authenticationService
}

// NewAuthenticationHandler builds an AuthenticationHandler.
func NewAuthenticationHandler(authentications authenticationService) *AuthenticationHandler {
	return &AuthenticationHandler{authentications: authentications}
}

// Create handles POST /api/v1/authentication-methods.
func (h *AuthenticationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req authenticationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.authentications.Create(r.Context(), req.toAuthentication(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newAuthenticationResponse(created))
}

// List handles GET /api/v1/authentication-methods.
func (h *AuthenticationHandler) List(w http.ResponseWriter, r *http.Request) {
	auths, err := h.authentications.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAuthenticationListResponse(auths))
}

// Get handles GET /api/v1/authentication-methods/{id}.
func (h *AuthenticationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	a, err := h.authentications.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAuthenticationResponse(a))
}

// Update handles PUT /api/v1/authentication-methods/{id}.
func (h *AuthenticationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req authenticationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.authentications.Update(r.Context(), req.toAuthentication(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAuthenticationResponse(updated))
}

// Delete handles DELETE /api/v1/authentication-methods/{id}.
func (h *AuthenticationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.authentications.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
