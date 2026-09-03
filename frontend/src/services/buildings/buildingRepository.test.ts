import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listBuildings, listBuildingsBySiteId, getBuildingById, createBuilding, updateBuilding, deleteBuilding } from './buildingRepository'

/** Mirrors oltRepository.test.ts's shape exactly -- no client-side search/sort/pagination, direct GET/:id since Building has its own Detail page. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function buildingDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'b1',
    site_id: 's1',
    name: 'Main Office',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listBuildings', () => {
  it('maps every Building from the DTO', async () => {
    apiFetch.mockResolvedValue({ buildings: [buildingDto({ id: 'b1' }), buildingDto({ id: 'b2' })] })

    const result = await listBuildings()

    expect(result.map((b) => b.id)).toEqual(['b1', 'b2'])
  })
})

describe('listBuildingsBySiteId', () => {
  it('returns only buildings belonging to the given site', async () => {
    apiFetch.mockResolvedValue({
      buildings: [
        buildingDto({ id: 'b1', site_id: 's1' }),
        buildingDto({ id: 'b2', site_id: 's2' }),
        buildingDto({ id: 'b3', site_id: 's1' }),
      ],
    })

    const result = await listBuildingsBySiteId('s1')

    expect(result.map((b) => b.id)).toEqual(['b1', 'b3'])
  })
})

describe('getBuildingById', () => {
  it('returns the building when found', async () => {
    apiFetch.mockResolvedValue(buildingDto({ id: 'b1' }))

    const result = await getBuildingById('b1')

    expect(result?.id).toBe('b1')
  })

  it('returns null instead of throwing when the building does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getBuildingById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getBuildingById('b1')).rejects.toThrow('boom')
  })
})

describe('createBuilding', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(buildingDto({ id: 'new' }))

    await createBuilding({ siteId: 's1', name: 'Main Office', description: 'HQ' })

    expect(apiFetch).toHaveBeenCalledWith('/buildings/', {
      method: 'POST',
      body: { site_id: 's1', name: 'Main Office', description: 'HQ' },
    })
  })
})

describe('updateBuilding', () => {
  it('sends a PUT with the request body in the API wire shape, passing through siteId unchanged', async () => {
    apiFetch.mockResolvedValue(buildingDto({ id: 'b1', name: 'Renamed' }))

    await updateBuilding('b1', { name: 'Renamed', description: 'Updated', siteId: 's1' })

    expect(apiFetch).toHaveBeenCalledWith('/buildings/b1', {
      method: 'PUT',
      body: { site_id: 's1', name: 'Renamed', description: 'Updated' },
    })
  })
})

describe('deleteBuilding', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteBuilding('b1')

    expect(apiFetch).toHaveBeenCalledWith('/buildings/b1', { method: 'DELETE' })
  })
})
