package event

import (
	"context"

	"github.com/google/uuid"
)

// EventRepository persists Events. There is deliberately no Update or
// Delete: events are immutable (see this package's doc comment) — once
// written, a record only ever gets read back, never changed or removed.
//
// Nothing in this package implements EventRepository — no SQL, no
// migrations — so the domain has zero dependency on any storage
// technology. A concrete implementation (internal/event/postgres)
// satisfies it.
type EventRepository interface {
	Create(ctx context.Context, e Event) (Event, error)
	ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]Event, error)
}
