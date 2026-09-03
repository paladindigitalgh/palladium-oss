// Package httpapi is the Workflow domain's REST layer, a direct port of
// the former internal/provisioning/httpapi with one addition: an
// /execute action route (see workflow_handler.go).
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
)

// instanceCreateRequest is the JSON body for POST
// /api/v1/workflow-instances. It carries only ServiceID and
// DefinitionName — Status, RetryCount, ErrorMessage, and the lifecycle
// timestamps are all forced by Service.Create regardless of what a
// caller supplies. RequestedByUserID is set from the authenticated
// caller's own identity, never from the request body.
type instanceCreateRequest struct {
	ServiceID      uuid.UUID `json:"service_id"`
	DefinitionName string    `json:"definition_name"`
}

// instanceResponse is the JSON representation of a WorkflowInstance
// returned to clients, decoupled from workflow.Instance's Go field
// layout.
type instanceResponse struct {
	ID                uuid.UUID  `json:"id"`
	DefinitionName    string     `json:"definition_name"`
	ServiceID         uuid.UUID  `json:"service_id"`
	RequestedByUserID *uuid.UUID `json:"requested_by_user_id"`
	Status            string     `json:"status"`
	RetryCount        int        `json:"retry_count"`
	ErrorMessage      *string    `json:"error_message"`
	StartedAt         *time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func newInstanceResponse(i workflow.Instance) instanceResponse {
	return instanceResponse{
		ID:                i.ID,
		DefinitionName:    i.DefinitionName,
		ServiceID:         i.ServiceID,
		RequestedByUserID: i.RequestedByUserID,
		Status:            string(i.Status),
		RetryCount:        i.RetryCount,
		ErrorMessage:      i.ErrorMessage,
		StartedAt:         i.StartedAt,
		CompletedAt:       i.CompletedAt,
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
}

// instanceListResponse wraps a slice of instances in an object rather
// than returning a bare JSON array, matching every other list response
// in this codebase.
type instanceListResponse struct {
	WorkflowInstances []instanceResponse `json:"workflow_instances"`
}

func newInstanceListResponse(instances []workflow.Instance) instanceListResponse {
	resp := instanceListResponse{WorkflowInstances: make([]instanceResponse, len(instances))}
	for i, instance := range instances {
		resp.WorkflowInstances[i] = newInstanceResponse(instance)
	}
	return resp
}
