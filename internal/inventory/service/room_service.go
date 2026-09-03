package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// RoomService is the Inventory domain's business logic for Rooms. See
// SiteService's doc comment for why this depends only on
// inventory.RoomRepository, not clock.Clock.
type RoomService struct {
	rooms inventory.RoomRepository
}

// NewRoomService builds a RoomService.
func NewRoomService(rooms inventory.RoomRepository) *RoomService {
	return &RoomService{rooms: rooms}
}

// Get retrieves a Room by ID.
func (s *RoomService) Get(ctx context.Context, id uuid.UUID) (inventory.Room, error) {
	return s.rooms.Get(ctx, id)
}

// List returns every Room.
func (s *RoomService) List(ctx context.Context) ([]inventory.Room, error) {
	return s.rooms.List(ctx)
}

// Create validates room and, if valid, persists it. See SiteService.Create
// for why validation happens here rather than elsewhere.
func (s *RoomService) Create(ctx context.Context, room inventory.Room) (inventory.Room, error) {
	if err := room.Validate(); err != nil {
		return inventory.Room{}, err
	}
	return s.rooms.Create(ctx, room)
}

// Update validates room and, if valid, persists the change. See
// SiteService.Create for why validation happens here rather than
// elsewhere.
func (s *RoomService) Update(ctx context.Context, room inventory.Room) (inventory.Room, error) {
	if err := room.Validate(); err != nil {
		return inventory.Room{}, err
	}
	return s.rooms.Update(ctx, room)
}

// Delete removes the Room identified by id.
func (s *RoomService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.rooms.Delete(ctx, id)
}
