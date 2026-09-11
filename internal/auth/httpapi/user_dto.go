package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
)

// This file's DTOs are separate from dto.go's loginRequest/loginResponse
// and use snake_case JSON (created_at, not createdAt) — matching
// internal/provider/httpapi/dto.go's convention, which is the one to
// follow here: loginResponse's camelCase is a documented one-off dictated
// by that endpoint's exact specification, not this package's general
// style.

// createUserRequest is the JSON body for POST /api/v1/users.
//
// It has no Status field: a newly created account always starts Active
// (see service.UserManagementService.Create) — this is not a caller
// choice, so there is nothing here for a caller to set. FirstName/
// LastName are both optional (see auth.User's doc comment) — a caller
// may leave either or both blank.
type createUserRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

// updateUserRoleRequest is the JSON body for PUT /api/v1/users/{id}/role.
type updateUserRoleRequest struct {
	Role string `json:"role"`
}

// userResponse is the JSON representation of a User returned to clients.
// PasswordHash never appears here, or anywhere in this package.
type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newUserResponse(u auth.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
		Status:    string(u.Status),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// userListResponse wraps a slice of users in an object rather than
// returning a bare JSON array — the same reasoning as
// internal/provider/httpapi's providerListResponse.
type userListResponse struct {
	Users []userResponse `json:"users"`
}

func newUserListResponse(users []auth.User) userListResponse {
	resp := userListResponse{Users: make([]userResponse, len(users))}
	for i, u := range users {
		resp.Users[i] = newUserResponse(u)
	}
	return resp
}
