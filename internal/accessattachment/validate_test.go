package accessattachment_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/serviceequipment/validate_test.go's
// helper of the same name: every domain package's Validate() must
// return an *apperror.Error of KindInvalid.
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

func validAccessAttachment() accessattachment.AccessAttachment {
	return accessattachment.AccessAttachment{
		AccessInterfaceID:  uuid.New(),
		ServiceEquipmentID: uuid.New(),
	}
}

func TestAccessAttachmentValidate(t *testing.T) {
	if err := validAccessAttachment().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, accessattachment.AccessAttachment{}.Validate())
}

func TestAccessAttachmentValidateRequiresAccessInterfaceID(t *testing.T) {
	a := validAccessAttachment()
	a.AccessInterfaceID = uuid.Nil

	assertInvalid(t, a.Validate())
}

func TestAccessAttachmentValidateRequiresServiceEquipmentID(t *testing.T) {
	a := validAccessAttachment()
	a.ServiceEquipmentID = uuid.Nil

	assertInvalid(t, a.Validate())
}

func TestAccessAttachmentValidateRemovalReasonIsOptional(t *testing.T) {
	a := validAccessAttachment() // no removal reason set
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (no removal reason) = %v, want nil", err)
	}

	a.RemovalReason = "Customer disconnected service"
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (with removal reason) = %v, want nil", err)
	}
}

func TestAccessAttachmentValidateLifecycleTimestampsAreOptional(t *testing.T) {
	a := validAccessAttachment() // no lifecycle timestamps set
	if err := a.Validate(); err != nil {
		t.Errorf("Validate() (no lifecycle timestamps) = %v, want nil", err)
	}
}

func TestAccessAttachmentActive(t *testing.T) {
	a := validAccessAttachment()
	if !a.Active() {
		t.Error("Active() = false for an AccessAttachment with no RemovedAt, want true")
	}

	removedAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	a.RemovedAt = &removedAt
	if a.Active() {
		t.Error("Active() = true for an AccessAttachment with RemovedAt set, want false")
	}
}
