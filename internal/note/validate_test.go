package note_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/note"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/contact/validate_test.go's helper of the
// same name: every domain package's Validate() must return an
// *apperror.Error of KindInvalid.
func assertInvalid(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("Validate() error is not an *apperror.Error: %v", err)
	}
	if appErr.Kind != apperror.KindInvalid {
		t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindInvalid)
	}
}

func validNote() note.Note {
	return note.Note{
		EntityType:   "customer",
		EntityID:     uuid.New(),
		AuthorUserID: uuid.New(),
		AuthorEmail:  "operator@example.com",
		Body:         "Called the customer back, issue resolved.",
	}
}

func TestNoteValidate(t *testing.T) {
	if err := validNote().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, note.Note{}.Validate())
}

func TestNoteValidateRequiresEntityType(t *testing.T) {
	n := validNote()
	n.EntityType = ""

	assertInvalid(t, n.Validate())
}

func TestNoteValidateRequiresEntityID(t *testing.T) {
	n := validNote()
	n.EntityID = uuid.Nil

	assertInvalid(t, n.Validate())
}

func TestNoteValidateRequiresAuthorUserID(t *testing.T) {
	n := validNote()
	n.AuthorUserID = uuid.Nil

	assertInvalid(t, n.Validate())
}

func TestNoteValidateRequiresBody(t *testing.T) {
	n := validNote()
	n.Body = "   "

	assertInvalid(t, n.Validate())
}

func TestNoteValidateEntityTypeIsNotRestrictedToAKnownSet(t *testing.T) {
	// Deliberately unlike ContactRole/ContactStatus: EntityType is a
	// loose reference (see model.go's doc comment), so any non-blank
	// value is accepted, not just "customer"/"device"/"service".
	n := validNote()
	n.EntityType = "site"

	if err := n.Validate(); err != nil {
		t.Errorf("Validate() (entity_type = site) = %v, want nil", err)
	}
}
