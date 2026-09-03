// Package postgres implements the Event domain's EventRepository against
// PostgreSQL using pgx directly — no ORM — following the pattern
// established by internal/serviceequipment/postgres.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// EventRepository implements event.EventRepository against PostgreSQL.
type EventRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ event.EventRepository = (*EventRepository)(nil)

// NewEventRepository builds an EventRepository.
func NewEventRepository(db database.Querier, clock clock.Clock, ids id.Generator) *EventRepository {
	return &EventRepository{db: db, clock: clock, ids: ids}
}

// Create inserts e and returns the persisted record. ID and CreatedAt are
// assigned by the repository, mirroring every other repository's Create
// in this codebase.
func (r *EventRepository) Create(ctx context.Context, e event.Event) (event.Event, error) {
	const query = `
		INSERT INTO events (id, entity_type, entity_id, type, message, metadata, actor_user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, entity_type, entity_id, type, message, metadata, actor_user_id, created_at
	`

	metadata, err := marshalMetadata(e.Metadata)
	if err != nil {
		return event.Event{}, apperror.Invalid("metadata must be JSON-serializable")
	}

	created, err := scanEvent(r.db.QueryRow(ctx, query,
		r.ids.New(), e.EntityType, e.EntityID, e.Type, e.Message, metadata, e.ActorUserID, r.clock.Now()))
	if err != nil {
		return event.Event{}, translateError("create event", err)
	}
	return created, nil
}

// ListByEntity returns every Event recorded for (entityType, entityID),
// oldest first — the natural order for a Timeline section to render
// chronologically.
func (r *EventRepository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]event.Event, error) {
	const query = `
		SELECT id, entity_type, entity_id, type, message, metadata, actor_user_id, created_at
		FROM events
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query, entityType, entityID)
	if err != nil {
		return nil, translateError("list events by entity", err)
	}
	defer rows.Close()

	events := []event.Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, translateError("scan event row", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list events by entity", err)
	}

	return events, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row rowScanner) (event.Event, error) {
	var (
		e        event.Event
		metadata []byte
	)
	err := row.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.Type, &e.Message, &metadata, &e.ActorUserID, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return event.Event{}, err
		}
		return event.Event{}, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &e.Metadata); err != nil {
			return event.Event{}, fmt.Errorf("unmarshal event metadata: %w", err)
		}
	}
	return e, nil
}

func marshalMetadata(m map[string]any) ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}
