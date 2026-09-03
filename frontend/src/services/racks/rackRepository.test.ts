import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listRacks, listRacksByRoomId, getRackById, createRack, updateRack, deleteRack } from './rackRepository'

/** Mirrors roomRepository.test.ts's shape exactly -- no client-side search/sort/pagination, direct GET/:id since Rack has its own Detail page. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function rackDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'rk1',
    room_id: 'r1',
    name: 'Rack A',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listRacks', () => {
  it('maps every Rack from the DTO, including a null room_id', async () => {
    apiFetch.mockResolvedValue({ racks: [rackDto({ id: 'rk1' }), rackDto({ id: 'rk2', room_id: null })] })

    const result = await listRacks()

    expect(result.map((r) => r.id)).toEqual(['rk1', 'rk2'])
    expect(result[1].roomId).toBeNull()
  })
})

describe('listRacksByRoomId', () => {
  it('returns only racks belonging to the given room', async () => {
    apiFetch.mockResolvedValue({
      racks: [
        rackDto({ id: 'rk1', room_id: 'r1' }),
        rackDto({ id: 'rk2', room_id: 'r2' }),
        rackDto({ id: 'rk3', room_id: null }),
        rackDto({ id: 'rk4', room_id: 'r1' }),
      ],
    })

    const result = await listRacksByRoomId('r1')

    expect(result.map((r) => r.id)).toEqual(['rk1', 'rk4'])
  })
})

describe('getRackById', () => {
  it('returns the rack when found', async () => {
    apiFetch.mockResolvedValue(rackDto({ id: 'rk1' }))

    const result = await getRackById('rk1')

    expect(result?.id).toBe('rk1')
  })

  it('returns null instead of throwing when the rack does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getRackById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getRackById('rk1')).rejects.toThrow('boom')
  })
})

describe('createRack', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(rackDto({ id: 'new' }))

    await createRack({ roomId: 'r1', name: 'Rack A', description: 'Row 1' })

    expect(apiFetch).toHaveBeenCalledWith('/racks/', {
      method: 'POST',
      body: { room_id: 'r1', name: 'Rack A', description: 'Row 1' },
    })
  })
})

describe('updateRack', () => {
  it('sends a PUT with the request body in the API wire shape, passing through roomId unchanged', async () => {
    apiFetch.mockResolvedValue(rackDto({ id: 'rk1', name: 'Renamed' }))

    await updateRack('rk1', { name: 'Renamed', description: 'Updated', roomId: 'r1' })

    expect(apiFetch).toHaveBeenCalledWith('/racks/rk1', {
      method: 'PUT',
      body: { room_id: 'r1', name: 'Renamed', description: 'Updated' },
    })
  })

  it('sends a null room_id through unchanged when the rack was never assigned', async () => {
    apiFetch.mockResolvedValue(rackDto({ id: 'rk1', room_id: null }))

    await updateRack('rk1', { name: 'Rack A', description: '', roomId: null })

    const [, init] = apiFetch.mock.calls[0]
    expect((init.body as { room_id: string | null }).room_id).toBeNull()
  })
})

describe('deleteRack', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteRack('rk1')

    expect(apiFetch).toHaveBeenCalledWith('/racks/rk1', { method: 'DELETE' })
  })
})
