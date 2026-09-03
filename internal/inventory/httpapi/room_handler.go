package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// roomService is the seam RoomHandler depends on instead of a concrete
// *service.RoomService. See siteService's doc comment for why.
type roomService interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Room, error)
	List(ctx context.Context) ([]inventory.Room, error)
	Create(ctx context.Context, room inventory.Room) (inventory.Room, error)
	Update(ctx context.Context, room inventory.Room) (inventory.Room, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// RoomHandler serves the Room REST endpoints:
//
//	POST   /api/v1/rooms
//	GET    /api/v1/rooms
//	GET    /api/v1/rooms/{id}
//	PUT    /api/v1/rooms/{id}
//	DELETE /api/v1/rooms/{id}
//
// It depends only on roomService — never a repository directly — so it
// has no knowledge of PostgreSQL, SQL, or any storage technology.
type RoomHandler struct {
	rooms roomService
}

// NewRoomHandler builds a RoomHandler.
func NewRoomHandler(rooms roomService) *RoomHandler {
	return &RoomHandler{rooms: rooms}
}

// Create handles POST /api/v1/rooms.
func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req roomRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.rooms.Create(r.Context(), req.toRoom(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newRoomResponse(created))
}

// List handles GET /api/v1/rooms.
func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.rooms.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newRoomListResponse(rooms))
}

// Get handles GET /api/v1/rooms/{id}.
func (h *RoomHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	room, err := h.rooms.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newRoomResponse(room))
}

// Update handles PUT /api/v1/rooms/{id}.
func (h *RoomHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req roomRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.rooms.Update(r.Context(), req.toRoom(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newRoomResponse(updated))
}

// Delete handles DELETE /api/v1/rooms/{id}.
func (h *RoomHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.rooms.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
