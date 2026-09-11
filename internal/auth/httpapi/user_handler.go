package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// userManagementService is the seam UserHandler depends on instead of a
// concrete *service.UserManagementService — the same reasoning
// internal/provider/httpapi's providerService interface documents: it
// lets handler tests exercise HTTP behavior against a fake, with no real
// service, repository, or database involved.
type userManagementService interface {
	List(ctx context.Context) ([]auth.User, error)
	Create(ctx context.Context, email, password, firstName, lastName string, role auth.Role) (auth.User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role auth.Role) (auth.User, error)
	Deactivate(ctx context.Context, id uuid.UUID) (auth.User, error)
	Reactivate(ctx context.Context, id uuid.UUID) (auth.User, error)
}

// UserHandler serves the User Management REST endpoints:
//
//	POST   /api/v1/users
//	GET    /api/v1/users
//	PUT    /api/v1/users/{id}/role
//	POST   /api/v1/users/{id}/deactivate
//	POST   /api/v1/users/{id}/reactivate
//
// Every one of these routes is mounted behind
// authz.Middleware.RequireUserManagement() alone (Administrator only, no
// Read/Write split — see that method's doc comment), so this handler does
// no authorization checking itself. It depends only on
// userManagementService — never a repository directly — and every method
// is a thin decode/delegate/translate, with no business logic: that is
// UserManagementService's job.
type UserHandler struct {
	users userManagementService
}

// NewUserHandler builds a UserHandler.
func NewUserHandler(users userManagementService) *UserHandler {
	return &UserHandler{users: users}
}

// Create handles POST /api/v1/users.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.users.Create(r.Context(), req.Email, req.Password, req.FirstName, req.LastName, auth.Role(req.Role))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newUserResponse(created))
}

// List handles GET /api/v1/users.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserListResponse(users))
}

// UpdateRole handles PUT /api/v1/users/{id}/role.
func (h *UserHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req updateUserRoleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.users.UpdateRole(r.Context(), id, auth.Role(req.Role))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserResponse(updated))
}

// Deactivate handles POST /api/v1/users/{id}/deactivate.
func (h *UserHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.users.Deactivate(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserResponse(updated))
}

// Reactivate handles POST /api/v1/users/{id}/reactivate.
func (h *UserHandler) Reactivate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.users.Reactivate(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserResponse(updated))
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
