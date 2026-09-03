package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// eventLister is the seam EventHandler depends on instead of a concrete
// event.EventRepository, so handler tests can exercise HTTP behavior
// against a fake.
type eventLister interface {
	ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]event.Event, error)
}

// EventHandler serves the Event domain's one REST endpoint:
//
//	GET /api/v1/events?entity_type=&entity_id=
//
// Both query parameters are required: an unscoped "every event ever
// recorded" listing has no legitimate UI use case (see
// docs/09-WORKSPACE-SPECIFICATIONS.md — Timeline sections always belong
// to one object) and would grow without bound.
type EventHandler struct {
	events eventLister
}

// NewEventHandler builds an EventHandler.
func NewEventHandler(events eventLister) *EventHandler {
	return &EventHandler{events: events}
}

// List handles GET /api/v1/events.
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		httpx.WriteError(w, apperror.Invalid("entity_type is required"))
		return
	}

	entityID, err := uuid.Parse(r.URL.Query().Get("entity_id"))
	if err != nil {
		httpx.WriteError(w, apperror.Invalid("entity_id must be a valid UUID"))
		return
	}

	events, err := h.events.ListByEntity(r.Context(), entityType, entityID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newEventListResponse(events))
}
