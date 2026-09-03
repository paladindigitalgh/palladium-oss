import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listSites, getSiteById, createSite, updateSite, deleteSite } from './siteRepository'

/**
 * Mirrors accessNetworkRepository.test.ts's shape, trimmed to what Site
 * actually supports: no status field, so no status-filter test.
 */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function siteDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 's1',
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

describe('listSites', () => {
  it('filters by search term across name and id', async () => {
    apiFetch.mockResolvedValue({ sites: [siteDto({ id: 's1', name: 'Main Office' }), siteDto({ id: 's2', name: 'Warehouse' })] })

    const result = await listSites({ search: 'main' })

    expect(result.items.map((s) => s.id)).toEqual(['s1'])
  })

  it('sorts by name ascending by default', async () => {
    apiFetch.mockResolvedValue({ sites: [siteDto({ id: 's1', name: 'Zeta' }), siteDto({ id: 's2', name: 'Alpha' })] })

    const result = await listSites()

    expect(result.items.map((s) => s.name)).toEqual(['Alpha', 'Zeta'])
  })

  it('reverses sort direction on request', async () => {
    apiFetch.mockResolvedValue({ sites: [siteDto({ id: 's1', name: 'Alpha' }), siteDto({ id: 's2', name: 'Zeta' })] })

    const result = await listSites({ sortDirection: 'desc' })

    expect(result.items.map((s) => s.name)).toEqual(['Zeta', 'Alpha'])
  })

  it('paginates results while reporting the true total', async () => {
    const sites = Array.from({ length: 20 }, (_, i) => siteDto({ id: `s${i}`, name: `Site ${i}` }))
    apiFetch.mockResolvedValue({ sites })

    const result = await listSites({ page: 2, pageSize: 15 })

    expect(result.total).toBe(20)
    expect(result.items).toHaveLength(5)
  })
})

describe('getSiteById', () => {
  it('returns the site when found', async () => {
    apiFetch.mockResolvedValue(siteDto({ id: 's1' }))

    const result = await getSiteById('s1')

    expect(result?.id).toBe('s1')
  })

  it('returns null instead of throwing when the site does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getSiteById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getSiteById('s1')).rejects.toThrow('boom')
  })
})

describe('createSite', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(siteDto({ id: 'new' }))

    await createSite({ name: 'Main Office', description: 'HQ' })

    expect(apiFetch).toHaveBeenCalledWith('/sites/', {
      method: 'POST',
      body: { name: 'Main Office', description: 'HQ' },
    })
  })
})

describe('updateSite', () => {
  it('sends a PUT with the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(siteDto({ id: 's1', name: 'Renamed' }))

    await updateSite('s1', { name: 'Renamed', description: 'Updated' })

    expect(apiFetch).toHaveBeenCalledWith('/sites/s1', {
      method: 'PUT',
      body: { name: 'Renamed', description: 'Updated' },
    })
  })
})

describe('deleteSite', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteSite('s1')

    expect(apiFetch).toHaveBeenCalledWith('/sites/s1', { method: 'DELETE' })
  })
})
