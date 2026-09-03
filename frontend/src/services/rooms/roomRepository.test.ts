import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listRooms, listRoomsByBuildingId, getRoomById, createRoom, updateRoom, deleteRoom } from './roomRepository'

/** Mirrors buildingRepository.test.ts's shape exactly -- no client-side search/sort/pagination, direct GET/:id since Room has its own Detail page. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function roomDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'r1',
    building_id: 'b1',
    name: 'First Floor',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listRooms', () => {
  it('maps every Room from the DTO', async () => {
    apiFetch.mockResolvedValue({ rooms: [roomDto({ id: 'r1' }), roomDto({ id: 'r2' })] })

    const result = await listRooms()

    expect(result.map((r) => r.id)).toEqual(['r1', 'r2'])
  })
})

describe('listRoomsByBuildingId', () => {
  it('returns only rooms belonging to the given building', async () => {
    apiFetch.mockResolvedValue({
      rooms: [
        roomDto({ id: 'r1', building_id: 'b1' }),
        roomDto({ id: 'r2', building_id: 'b2' }),
        roomDto({ id: 'r3', building_id: 'b1' }),
      ],
    })

    const result = await listRoomsByBuildingId('b1')

    expect(result.map((r) => r.id)).toEqual(['r1', 'r3'])
  })
})

describe('getRoomById', () => {
  it('returns the room when found', async () => {
    apiFetch.mockResolvedValue(roomDto({ id: 'r1' }))

    const result = await getRoomById('r1')

    expect(result?.id).toBe('r1')
  })

  it('returns null instead of throwing when the room does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getRoomById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getRoomById('r1')).rejects.toThrow('boom')
  })
})

describe('createRoom', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(roomDto({ id: 'new' }))

    await createRoom({ buildingId: 'b1', name: 'First Floor', description: 'Main floor' })

    expect(apiFetch).toHaveBeenCalledWith('/rooms/', {
      method: 'POST',
      body: { building_id: 'b1', name: 'First Floor', description: 'Main floor' },
    })
  })
})

describe('updateRoom', () => {
  it('sends a PUT with the request body in the API wire shape, passing through buildingId unchanged', async () => {
    apiFetch.mockResolvedValue(roomDto({ id: 'r1', name: 'Renamed' }))

    await updateRoom('r1', { name: 'Renamed', description: 'Updated', buildingId: 'b1' })

    expect(apiFetch).toHaveBeenCalledWith('/rooms/r1', {
      method: 'PUT',
      body: { building_id: 'b1', name: 'Renamed', description: 'Updated' },
    })
  })
})

describe('deleteRoom', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteRoom('r1')

    expect(apiFetch).toHaveBeenCalledWith('/rooms/r1', { method: 'DELETE' })
  })
})
