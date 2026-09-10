package note

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether n has every required field set: a present
// EntityType and EntityID (what this Note is about), a present
// AuthorUserID (who wrote it — see model.go's doc comment on why this is
// always required, unlike Event's nullable ActorUserID), and a non-blank
// Body. AuthorEmail is never checked here: it is captured automatically
// from the caller's JWT claims (see httpapi's Create handler), never
// supplied by the request body itself, so there is nothing for a client
// to get wrong about it.
func (n Note) Validate() error {
	errs := validate.New()

	if !validate.Required(n.EntityType) {
		errs.Add("entity_type", "is required")
	}
	if n.EntityID == uuid.Nil {
		errs.Add("entity_id", "is required")
	}
	if n.AuthorUserID == uuid.Nil {
		errs.Add("author_user_id", "is required")
	}
	if !validate.Required(n.Body) {
		errs.Add("body", "is required")
	}

	return errs.Err()
}
