package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// profileService is the seam ProfileHandler depends on instead of a
// concrete *service.ProfileService — the same reasoning userManagementService
// documents, applied to the self-service half of User Management.
type profileService interface {
	Get(ctx context.Context, id uuid.UUID) (auth.User, error)
	UpdateName(ctx context.Context, id uuid.UUID, firstName, lastName string) (auth.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) (auth.User, error)
}

// ProfileHandler serves the signed-in caller's own account:
//
//	GET /api/v1/me
//	PUT /api/v1/me
//	PUT /api/v1/me/password
//
// Every one of these routes is mounted behind only auth.Middleware (see
// internal/server/router.go) — no capability gate — since every Role,
// not just Administrator, may view and edit their own account. This is
// the one deliberate way this differs from UserHandler: /users acts on a
// User given by ID in the path (Administrator only), /me always acts on
// whichever User the caller's own JWT claims name.
type ProfileHandler struct {
	profile profileService
}

// NewProfileHandler builds a ProfileHandler.
func NewProfileHandler(profile profileService) *ProfileHandler {
	return &ProfileHandler{profile: profile}
}

// Get handles GET /api/v1/me.
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := callerID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	user, err := h.profile.Get(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserResponse(user))
}

// UpdateName handles PUT /api/v1/me.
func (h *ProfileHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	userID, err := callerID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req updateProfileNameRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.profile.UpdateName(r.Context(), userID, req.FirstName, req.LastName)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserResponse(updated))
}

// ChangePassword handles PUT /api/v1/me/password.
func (h *ProfileHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := callerID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req changePasswordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.profile.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserResponse(updated))
}

// callerID reads the authenticated caller's own User ID from their JWT
// claims. Every route ProfileHandler serves is mounted behind
// auth.Middleware (see its own doc comment), so claims are always present
// in practice; the explicit apperror.KindUnauthorized fallback exists so
// a missing context value fails loudly and safely rather than silently
// acting on the zero uuid.UUID.
func callerID(r *http.Request) (uuid.UUID, error) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.Unauthorized("authentication required")
	}
	return claims.UserID, nil
}
