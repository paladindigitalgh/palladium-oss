// Package httpapi is the Event domain's REST layer. It exposes read-only
// access to Events — there is no create route, since events are written
// internally by domain/workflow code, never posted by a client (see
// internal/event's package doc comment).
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/event"
)

// eventResponse is the JSON representation of an Event returned to
// clients, decoupled from event.Event's Go field layout.
type eventResponse struct {
	ID          uuid.UUID      `json:"id"`
	EntityType  string         `json:"entity_type"`
	EntityID    uuid.UUID      `json:"entity_id"`
	Type        string         `json:"type"`
	Message     string         `json:"message"`
	Metadata    map[string]any `json:"metadata"`
	ActorUserID *uuid.UUID     `json:"actor_user_id"`
	CreatedAt   time.Time      `json:"created_at"`
}

func newEventResponse(e event.Event) eventResponse {
	return eventResponse{
		ID:          e.ID,
		EntityType:  e.EntityType,
		EntityID:    e.EntityID,
		Type:        e.Type,
		Message:     e.Message,
		Metadata:    e.Metadata,
		ActorUserID: e.ActorUserID,
		CreatedAt:   e.CreatedAt,
	}
}

// eventListResponse wraps a slice of events in an object rather than
// returning a bare JSON array, matching every other list response in
// this codebase.
type eventListResponse struct {
	Events []eventResponse `json:"events"`
}

func newEventListResponse(events []event.Event) eventListResponse {
	resp := eventListResponse{Events: make([]eventResponse, len(events))}
	for i, e := range events {
		resp.Events[i] = newEventResponse(e)
	}
	return resp
}
