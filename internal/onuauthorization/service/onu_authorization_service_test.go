package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeRepository is an in-memory onuauthorization.Repository.
type fakeRepository struct {
	created onuauthorization.OnuAuthorization
	active  onuauthorization.OnuAuthorization
	err     error

	createCalled bool
	updateCalled bool
	gotUpdate    onuauthorization.OnuAuthorization
}

func (f *fakeRepository) Create(_ context.Context, a onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error) {
	f.createCalled = true
	if f.err != nil {
		return onuauthorization.OnuAuthorization{}, f.err
	}
	return f.created, nil
}

func (f *fakeRepository) Update(_ context.Context, a onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error) {
	f.updateCalled = true
	f.gotUpdate = a
	if f.err != nil {
		return onuauthorization.OnuAuthorization{}, f.err
	}
	return a, nil
}

func (f *fakeRepository) GetActiveByDeviceID(_ context.Context, _ uuid.UUID) (onuauthorization.OnuAuthorization, error) {
	if f.err != nil {
		return onuauthorization.OnuAuthorization{}, f.err
	}
	return f.active, nil
}

func TestOnuAuthorizationServiceCreateRejectsInvalidWithoutPersisting(t *testing.T) {
	repo := &fakeRepository{}
	s := NewOnuAuthorizationService(repo)

	_, err := s.Create(context.Background(), onuauthorization.OnuAuthorization{})
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("Create was called on the repository despite invalid input")
	}
}

func TestOnuAuthorizationServiceCreateSucceeds(t *testing.T) {
	deviceID, oltID := uuid.New(), uuid.New()
	want := onuauthorization.OnuAuthorization{ID: uuid.New(), DeviceID: deviceID, OLTID: oltID, Interface: "xgs/1/3", AuthorizedAt: time.Now()}
	repo := &fakeRepository{created: want}
	s := NewOnuAuthorizationService(repo)

	got, err := s.Create(context.Background(), onuauthorization.OnuAuthorization{
		DeviceID: deviceID, OLTID: oltID, Interface: "xgs/1/3", AuthorizedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}
	if !repo.createCalled {
		t.Error("Create was not called on the repository")
	}
}

func TestOnuAuthorizationServiceUpdateRejectsInvalidWithoutPersisting(t *testing.T) {
	repo := &fakeRepository{}
	s := NewOnuAuthorizationService(repo)

	_, err := s.Update(context.Background(), onuauthorization.OnuAuthorization{})
	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("Update was called on the repository despite invalid input")
	}
}

func TestOnuAuthorizationServiceUpdateSucceeds(t *testing.T) {
	repo := &fakeRepository{}
	s := NewOnuAuthorizationService(repo)

	now := time.Now()
	a := onuauthorization.OnuAuthorization{
		ID: uuid.New(), DeviceID: uuid.New(), OLTID: uuid.New(), Interface: "xgs/1/3",
		AuthorizedAt: now.Add(-time.Hour), DeauthorizedAt: &now,
	}
	got, err := s.Update(context.Background(), a)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if got.DeauthorizedAt == nil {
		t.Error("DeauthorizedAt was not preserved")
	}
	if !repo.updateCalled {
		t.Error("Update was not called on the repository")
	}
	if repo.gotUpdate.ID != a.ID {
		t.Errorf("Update called with ID %v, want %v", repo.gotUpdate.ID, a.ID)
	}
}

func TestOnuAuthorizationServiceGetActiveByDeviceID(t *testing.T) {
	deviceID := uuid.New()
	want := onuauthorization.OnuAuthorization{ID: uuid.New(), DeviceID: deviceID}
	repo := &fakeRepository{active: want}
	s := NewOnuAuthorizationService(repo)

	got, err := s.GetActiveByDeviceID(context.Background(), deviceID)
	if err != nil {
		t.Fatalf("GetActiveByDeviceID() = %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}
}

func TestOnuAuthorizationServiceGetActiveByDeviceIDPropagatesNotFound(t *testing.T) {
	repo := &fakeRepository{err: apperror.NotFound("no active onu authorization for device")}
	s := NewOnuAuthorizationService(repo)

	_, err := s.GetActiveByDeviceID(context.Background(), uuid.New())
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
