import type { Room } from '@/types/room'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real Room data source. GET /rooms has no server-side filtering
 * (see internal/inventory/httpapi), so "a building's Rooms" is resolved
 * by fetching every Room once and filtering client-side -- the same
 * pattern buildingRepository.ts uses. Room has its own Detail page, so
 * getRoomById hits GET /rooms/:id directly.
 */

interface RoomDto {
  id: string
  building_id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: RoomDto): Room {
  return {
    id: dto.id,
    buildingId: dto.building_id,
    name: dto.name,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listRooms(): Promise<Room[]> {
  const { rooms } = await apiFetch<{ rooms: RoomDto[] }>('/rooms/')
  return rooms.map(fromDto)
}

export async function listRoomsByBuildingId(buildingId: string): Promise<Room[]> {
  const rooms = await listRooms()
  return rooms.filter((room) => room.buildingId === buildingId)
}

/** Fetches a single Room, returning null (not throwing) when it does not exist. */
export async function getRoomById(id: string): Promise<Room | null> {
  try {
    const dto = await apiFetch<RoomDto>(`/rooms/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateRoomInput {
  buildingId: string
  name: string
  description: string
}

export async function createRoom(input: CreateRoomInput): Promise<Room> {
  const dto = await apiFetch<RoomDto>('/rooms/', {
    method: 'POST',
    body: { building_id: input.buildingId, name: input.name, description: input.description },
  })
  return fromDto(dto)
}

export interface UpdateRoomInput {
  name: string
  description: string
  /**
   * Not user-editable (see RoomFormDialog.vue -- no building picker
   * exists in this workspace). Callers pass the Room's current
   * buildingId through unchanged: PUT replaces every mutable column
   * (see internal/inventory/postgres's Update), so omitting it here
   * would silently move the Room to no building.
   */
  buildingId: string
}

export async function updateRoom(id: string, input: UpdateRoomInput): Promise<Room> {
  const dto = await apiFetch<RoomDto>(`/rooms/${id}`, {
    method: 'PUT',
    body: { building_id: input.buildingId, name: input.name, description: input.description },
  })
  return fromDto(dto)
}

/**
 * Deletes the Room identified by id. rooms.id is referenced by
 * racks.room_id ON DELETE RESTRICT, so this throws an ApiError with kind
 * "conflict" if the room still has any Rack.
 */
export async function deleteRoom(id: string): Promise<void> {
  await apiFetch<void>(`/rooms/${id}`, { method: 'DELETE' })
}
