package httpapi

import (
	"context"
	"net/http"
	"strconv"

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
	ListRecent(ctx context.Context, limit int) ([]event.Event, error)
}

// EventHandler serves the Event domain's two REST endpoints:
//
//	GET /api/v1/events?entity_type=&entity_id=
//	GET /api/v1/events/recent?limit=
//
// List's two query parameters are both required: an unbounded "every
// event ever recorded for this entity_type, but no entity_id filter"
// listing has no legitimate UI use case (see
// docs/09-WORKSPACE-SPECIFICATIONS.md — Timeline sections always belong
// to one object) and would grow without bound. ListRecent is a
// different, deliberately bounded shape — the Dashboard's system-wide
// activity feed only ever wants the most recent handful of events, not
// an unscoped dump — so it needs no entity filter at all, only a capped
// limit.
type EventHandler struct {
	events eventLister
}

// defaultRecentLimit and maxRecentLimit bound ListRecent's limit query
// parameter: a caller asking for nothing gets a sensible default, and a
// caller asking for an excessive amount is silently capped rather than
// rejected — the same "clamp, don't 422" treatment as any other
// unauthenticated-adjacent input that has an obviously reasonable
// ceiling.
const (
	defaultRecentLimit = 20
	maxRecentLimit     = 100
)

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

// ListRecent handles GET /api/v1/events/recent.
func (h *EventHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	events, err := h.events.ListRecent(r.Context(), recentLimit(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newEventListResponse(events))
}

// recentLimit parses the "limit" query parameter, falling back to
// defaultRecentLimit when it is absent, unparseable, or not a positive
// integer, and clamping anything above maxRecentLimit down to it rather
// than rejecting the request — see the constants' own doc comment.
func recentLimit(r *http.Request) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultRecentLimit
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return defaultRecentLimit
	}
	if limit > maxRecentLimit {
		return maxRecentLimit
	}
	return limit
}
