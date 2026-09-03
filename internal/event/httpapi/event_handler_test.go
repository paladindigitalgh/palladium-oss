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

// stubEventLister's methods are all pointer receivers (not just
// ListRecent, which needs one to record gotLimit) so every test below
// constructs it the same way, &stubEventLister{...}.
type stubEventLister struct {
	events []event.Event
	err    error

	// gotLimit records the limit ListRecent was last called with, so
	// tests can assert on what the handler actually parsed from the
	// query string without needing a real repository behind it.
	gotLimit int
}

func (s *stubEventLister) ListByEntity(context.Context, string, uuid.UUID) ([]event.Event, error) {
	return s.events, s.err
}

func (s *stubEventLister) ListRecent(_ context.Context, limit int) ([]event.Event, error) {
	s.gotLimit = limit
	return s.events, s.err
}

func TestListRequiresEntityType(t *testing.T) {
	h := httpapi.NewEventHandler(&stubEventLister{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?entity_id="+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestListRequiresValidEntityID(t *testing.T) {
	h := httpapi.NewEventHandler(&stubEventLister{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?entity_type=service&entity_id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestListReturnsEvents(t *testing.T) {
	entityID := uuid.New()
	h := httpapi.NewEventHandler(&stubEventLister{events: []event.Event{
		{ID: uuid.New(), EntityType: "service", EntityID: entityID, Type: "workflow.started", Message: "Workflow started"},
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?entity_type=service&entity_id="+entityID.String(), nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestListRecentReturnsEvents mirrors TestListReturnsEvents: ListRecent
// needs no entity_type/entity_id at all, unlike List.
func TestListRecentReturnsEvents(t *testing.T) {
	lister := &stubEventLister{events: []event.Event{
		{ID: uuid.New(), EntityType: "service", EntityID: uuid.New(), Type: "workflow.started", Message: "Workflow started"},
	}}
	h := httpapi.NewEventHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/recent", nil)
	rec := httptest.NewRecorder()
	h.ListRecent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestListRecentDefaultsLimitWhenAbsent(t *testing.T) {
	lister := &stubEventLister{}
	h := httpapi.NewEventHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/recent", nil)
	h.ListRecent(httptest.NewRecorder(), req)

	if lister.gotLimit != 20 {
		t.Errorf("gotLimit = %d, want 20 (the default)", lister.gotLimit)
	}
}

func TestListRecentHonorsExplicitLimit(t *testing.T) {
	lister := &stubEventLister{}
	h := httpapi.NewEventHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/recent?limit=5", nil)
	h.ListRecent(httptest.NewRecorder(), req)

	if lister.gotLimit != 5 {
		t.Errorf("gotLimit = %d, want 5", lister.gotLimit)
	}
}

func TestListRecentClampsExcessiveLimit(t *testing.T) {
	lister := &stubEventLister{}
	h := httpapi.NewEventHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/recent?limit=99999", nil)
	rec := httptest.NewRecorder()
	h.ListRecent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; an excessive limit should be clamped, not rejected; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if lister.gotLimit != 100 {
		t.Errorf("gotLimit = %d, want 100 (the max)", lister.gotLimit)
	}
}

func TestListRecentFallsBackOnUnparseableLimit(t *testing.T) {
	lister := &stubEventLister{}
	h := httpapi.NewEventHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/recent?limit=not-a-number", nil)
	rec := httptest.NewRecorder()
	h.ListRecent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; an unparseable limit should fall back to the default, not error; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if lister.gotLimit != 20 {
		t.Errorf("gotLimit = %d, want 20 (the default)", lister.gotLimit)
	}
}

func TestListRecentFallsBackOnNonPositiveLimit(t *testing.T) {
	lister := &stubEventLister{}
	h := httpapi.NewEventHandler(lister)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/recent?limit=0", nil)
	h.ListRecent(httptest.NewRecorder(), req)

	if lister.gotLimit != 20 {
		t.Errorf("gotLimit = %d, want 20 (the default)", lister.gotLimit)
	}
}
