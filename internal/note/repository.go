package note

import (
	"context"

	"github.com/google/uuid"
)

// NoteRepository persists Notes. There is deliberately no Update or
// Delete: Notes are immutable operational history (see this package's
// own doc comment) — once written, a record only ever gets read back,
// never changed or removed, the identical reasoning
// internal/event.EventRepository documents for itself.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/note/postgres) satisfies it.
type NoteRepository interface {
	Create(ctx context.Context, n Note) (Note, error)

	// ListByEntity returns every Note recorded for one entity, newest
	// first. This is the opposite order from Event's own ListByEntity
	// (oldest first, reconstructing a narrative timeline): a Notes
	// section is about surfacing the latest operator remark first, the
	// same "most recent status at a glance" reasoning
	// event.EventRepository.ListRecent documents for its own
	// newest-first ordering.
	ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]Note, error)
}
