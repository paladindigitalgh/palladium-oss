// Package note models Palladium's Note domain: free-text, operator-
// authored commentary attached to an entity elsewhere in the system (a
// Customer, Device, or Service today). EntityType/EntityID identify what
// a Note is about the same way internal/event's Event does — a loose
// reference, not a typed foreign key — so this package has no dependency
// on customer, inventory, service, or any other domain, and nothing
// stops a future caller from attaching a Note to some other entity
// without this package growing a new field or import for it.
//
// Unlike Event, every Note has a human author and is always written by a
// client request — never generated internally by workflow or domain
// code. That is the one deliberate difference from Event's shape, and
// why this is its own small package rather than an addition to
// internal/event: that package's own doc comment states "there is no
// create route ... never posted by a client," which a user-submitted
// Note flatly contradicts.
//
// A Note is immutable once written, the same "operational history, never
// updated or deleted" reasoning Event's own doc comment gives
// (docs/02-DESIGN-PRINCIPLES.md principle 10) — there is deliberately no
// Update or Delete anywhere in this package.
package note

import (
	"time"

	"github.com/google/uuid"
)

// Note is a single, timestamped, author-attributed remark about an
// entity elsewhere in the system.
type Note struct {
	ID         uuid.UUID
	EntityType string
	EntityID   uuid.UUID

	// AuthorUserID/AuthorEmail are both captured from the authenticated
	// caller's JWT claims at creation time (see httpapi's Create
	// handler), never supplied by the request body. AuthorEmail is a
	// deliberate snapshot, not resolved via a live join against
	// auth.User at read time — a Note keeps showing who wrote it even if
	// that User's email later changes or the account is deactivated, and
	// every Role that can read Notes can see who left one without also
	// needing internal/auth's User Management permission (see
	// internal/authz's RequireUserManagement gate on GET /users) just to
	// resolve a name.
	//
	// AuthorFirstName/AuthorLastName are the same kind of snapshot,
	// captured from the author's User record as it stood at creation
	// time (see httpapi's Create handler) rather than from JWT claims —
	// a JWT carries only ID and Email (see auth.Claims's doc comment),
	// deliberately not a display name that can change mid-session. Both
	// are optional, exactly as they are on auth.User: a Note written by
	// a User with no name set simply carries two empty strings here, and
	// callers fall back to AuthorEmail for display.
	AuthorUserID    uuid.UUID
	AuthorEmail     string
	AuthorFirstName string
	AuthorLastName  string

	Body string

	CreatedAt time.Time
}
