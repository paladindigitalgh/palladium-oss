// Package httpapi is the Provisioning domain's REST layer. It depends on
// internal/provisioning/service, never on a repository directly, and
// never exposes internal/provisioning's domain types over the wire — see
// the DTOs in this file. It mirrors internal/serviceequipment/httpapi
// exactly, with one addition: action sub-routes for the state machine
// (see provisioning_handler.go's doc comment for why).
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// provisioningJobCreateRequest is the JSON body for POST
// /api/v1/provisioning-jobs.
//
// It carries only ServiceID and Operation — not Status, RetryCount,
// ErrorMessage, StartedAt, CompletedAt, or RequestedByUserID. The first
// five are exactly the fields ProvisioningService.Create forces to their
// initial values regardless of what a caller supplies (see that method's
// doc comment), so accepting them here would invite a caller to believe
// they have an effect they do not; leaving them off the request type
// makes that impossible to even attempt, which is a stronger guarantee
// than validating them away. RequestedByUserID is deliberately not
// client-suppliable either: the handler sets it from the authenticated
// caller's own identity (see Create in provisioning_handler.go), not from
// the request body — a client naming an arbitrary RequestedByUserID would
// let anyone create a job "on behalf of" someone else.
type provisioningJobCreateRequest struct {
	ServiceID uuid.UUID `json:"service_id"`
	Operation string    `json:"operation"`
}

// provisioningJobFailRequest is the JSON body for POST
// /api/v1/provisioning-jobs/{id}/fail — the one transition that takes a
// caller-supplied value (goal 1's ErrorMessage) beyond the path
// parameter identifying which job.
type provisioningJobFailRequest struct {
	ErrorMessage string `json:"error_message"`
}

// provisioningJobResponse is the JSON representation of a
// ProvisioningJob returned to clients. Decoupling the wire format from
// ProvisioningJob's Go field layout and types means a change to how the
// domain model is composed internally can never silently change the
// API's JSON shape.
//
// Operation and Status are plain strings, not their domain enum types,
// for the same "DTOs only" separation
// internal/serviceequipment/httpapi.serviceEquipmentResponse documents.
// RequestedByUserID, ErrorMessage, StartedAt, and CompletedAt are left as
// their plain primitive pointer types (*uuid.UUID, *string, *time.Time):
// encoding/json already renders a nil pointer as JSON null, which is
// exactly goal 8's explicit instruction ("return nullable values as JSON
// null") — no special-casing needed here to get that behavior.
type provisioningJobResponse struct {
	ID                uuid.UUID  `json:"id"`
	ServiceID         uuid.UUID  `json:"service_id"`
	RequestedByUserID *uuid.UUID `json:"requested_by_user_id"`
	Operation         string     `json:"operation"`
	Status            string     `json:"status"`
	RetryCount        int        `json:"retry_count"`
	ErrorMessage      *string    `json:"error_message"`
	StartedAt         *time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func newProvisioningJobResponse(j provisioning.ProvisioningJob) provisioningJobResponse {
	return provisioningJobResponse{
		ID:                j.ID,
		ServiceID:         j.ServiceID,
		RequestedByUserID: j.RequestedByUserID,
		Operation:         string(j.Operation),
		Status:            string(j.Status),
		RetryCount:        j.RetryCount,
		ErrorMessage:      j.ErrorMessage,
		StartedAt:         j.StartedAt,
		CompletedAt:       j.CompletedAt,
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
	}
}

// provisioningJobListResponse wraps a slice of jobs in an object rather
// than returning a bare JSON array — the same reasoning as
// internal/serviceequipment/httpapi's serviceEquipmentListResponse: a
// bare top-level array can never gain sibling fields (a total count, a
// pagination cursor, ...) without becoming a breaking change for
// existing clients, while adding a field next to "provisioning_jobs" is
// not.
type provisioningJobListResponse struct {
	ProvisioningJobs []provisioningJobResponse `json:"provisioning_jobs"`
}

func newProvisioningJobListResponse(jobs []provisioning.ProvisioningJob) provisioningJobListResponse {
	resp := provisioningJobListResponse{ProvisioningJobs: make([]provisioningJobResponse, len(jobs))}
	for i, j := range jobs {
		resp.ProvisioningJobs[i] = newProvisioningJobResponse(j)
	}
	return resp
}
