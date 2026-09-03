package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/event/httpapi"
)

type stubEventLister struct {
	events []event.Event
	err    error
}

func (s stubEventLister) ListByEntity(context.Context, string, uuid.UUID) ([]event.Event, error) {
	return s.events, s.err
}

func TestListRequiresEntityType(t *testing.T) {
	h := httpapi.NewEventHandler(stubEventLister{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?entity_id="+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestListRequiresValidEntityID(t *testing.T) {
	h := httpapi.NewEventHandler(stubEventLister{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?entity_type=service&entity_id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestListReturnsEvents(t *testing.T) {
	entityID := uuid.New()
	h := httpapi.NewEventHandler(stubEventLister{events: []event.Event{
		{ID: uuid.New(), EntityType: "service", EntityID: entityID, Type: "workflow.started", Message: "Workflow started"},
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?entity_type=service&entity_id="+entityID.String(), nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
