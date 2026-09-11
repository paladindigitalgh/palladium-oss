// Package postgres implements the Note domain's NoteRepository against
// PostgreSQL using pgx directly — no ORM — following the pattern
// established by internal/event/postgres.
package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/note"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// NoteRepository implements note.NoteRepository against PostgreSQL.
type NoteRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ note.NoteRepository = (*NoteRepository)(nil)

// NewNoteRepository builds a NoteRepository.
func NewNoteRepository(db database.Querier, clock clock.Clock, ids id.Generator) *NoteRepository {
	return &NoteRepository{db: db, clock: clock, ids: ids}
}

// Create inserts n and returns the persisted record. ID and CreatedAt are
// assigned by the repository, mirroring every other repository's Create
// in this codebase.
func (r *NoteRepository) Create(ctx context.Context, n note.Note) (note.Note, error) {
	const query = `
		INSERT INTO notes (id, entity_type, entity_id, author_user_id, author_email, author_first_name, author_last_name, body, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, entity_type, entity_id, author_user_id, author_email, author_first_name, author_last_name, body, created_at
	`

	created, err := scanNote(r.db.QueryRow(ctx, query,
		r.ids.New(), n.EntityType, n.EntityID, n.AuthorUserID, n.AuthorEmail,
		n.AuthorFirstName, n.AuthorLastName, n.Body, r.clock.Now()))
	if err != nil {
		return note.Note{}, translateError("create note", err)
	}
	return created, nil
}

// ListByEntity returns every Note recorded for (entityType, entityID),
// newest first — see note.NoteRepository.ListByEntity's doc comment for
// why this order differs from Event's own ListByEntity.
func (r *NoteRepository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]note.Note, error) {
	const query = `
		SELECT id, entity_type, entity_id, author_user_id, author_email, author_first_name, author_last_name, body, created_at
		FROM notes
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, entityType, entityID)
	if err != nil {
		return nil, translateError("list notes by entity", err)
	}
	defer rows.Close()

	notes := []note.Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, translateError("scan note row", err)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list notes by entity", err)
	}

	return notes, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNote(row rowScanner) (note.Note, error) {
	var n note.Note
	err := row.Scan(&n.ID, &n.EntityType, &n.EntityID, &n.AuthorUserID, &n.AuthorEmail,
		&n.AuthorFirstName, &n.AuthorLastName, &n.Body, &n.CreatedAt)
	return n, err
}
