package httpapi

// updateProfileNameRequest is the JSON body for PUT /api/v1/me. Only
// FirstName/LastName can be changed here — Email, Role, and Status are
// deliberately absent: nothing yet needs a self-service email change (see
// internal/auth/repository.go), and a User can never grant themselves a
// different Role or Status (see internal/auth/service.UserManagementService,
// the only place either changes, gated behind RequireUserManagement).
type updateProfileNameRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// changePasswordRequest is the JSON body for PUT /api/v1/me/password.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
