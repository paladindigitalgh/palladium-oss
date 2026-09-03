import type { Rack } from '@/types/rack'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real Rack data source. GET /racks has no server-side filtering
 * (see internal/inventory/httpapi), so "a room's Racks" is resolved by
 * fetching every Rack once and filtering client-side -- the same
 * pattern roomRepository.ts uses. Rack has its own Detail page, so
 * getRackById hits GET /racks/:id directly. listRacks is also used
 * directly by DeviceFormDialog.vue's Rack picker.
 */

interface RackDto {
  id: string
  room_id: string | null
  name: string
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: RackDto): Rack {
  return {
    id: dto.id,
    roomId: dto.room_id,
    name: dto.name,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listRacks(): Promise<Rack[]> {
  const { racks } = await apiFetch<{ racks: RackDto[] }>('/racks/')
  return racks.map(fromDto)
}

export async function listRacksByRoomId(roomId: string): Promise<Rack[]> {
  const racks = await listRacks()
  return racks.filter((rack) => rack.roomId === roomId)
}

/** Fetches a single Rack, returning null (not throwing) when it does not exist. */
export async function getRackById(id: string): Promise<Rack | null> {
  try {
    const dto = await apiFetch<RackDto>(`/racks/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateRackInput {
  roomId: string
  name: string
  description: string
}

export async function createRack(input: CreateRackInput): Promise<Rack> {
  const dto = await apiFetch<RackDto>('/racks/', {
    method: 'POST',
    body: { room_id: input.roomId, name: input.name, description: input.description },
  })
  return fromDto(dto)
}

export interface UpdateRackInput {
  name: string
  description: string
  /**
   * Not user-editable (see RackFormDialog.vue -- no room picker exists
   * in this workspace). Callers pass the Rack's current roomId through
   * unchanged: PUT replaces every mutable column (see
   * internal/inventory/postgres's Update), so omitting it here would
   * silently unassign the Rack from its room.
   */
  roomId: string | null
}

export async function updateRack(id: string, input: UpdateRackInput): Promise<Rack> {
  const dto = await apiFetch<RackDto>(`/racks/${id}`, {
    method: 'PUT',
    body: { room_id: input.roomId, name: input.name, description: input.description },
  })
  return fromDto(dto)
}

/**
 * Deletes the Rack identified by id. racks.id is referenced by
 * devices.rack_id ON DELETE RESTRICT (database/migrations/00006), so
 * this throws an ApiError with kind "conflict" if the rack still has
 * any Device.
 */
export async function deleteRack(id: string): Promise<void> {
  await apiFetch<void>(`/racks/${id}`, { method: 'DELETE' })
}
