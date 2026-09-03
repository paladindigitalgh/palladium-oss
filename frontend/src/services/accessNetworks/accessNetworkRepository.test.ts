import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import {
  listAccessNetworks,
  getAccessNetworkById,
  createAccessNetwork,
  updateAccessNetwork,
  deleteAccessNetwork,
} from './accessNetworkRepository'

/** Mirrors customerRepository.test.ts's shape exactly -- same client-side search/filter/sort/pagination logic, same top-level repository shape. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function accessNetworkDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'an1',
    name: 'Metro North',
    status: 'Active',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listAccessNetworks', () => {
  it('filters by search term across name and id', async () => {
    apiFetch.mockResolvedValue({
      access_networks: [accessNetworkDto({ id: 'an1', name: 'Metro North' }), accessNetworkDto({ id: 'an2', name: 'Suburban West' })],
    })

    const result = await listAccessNetworks({ search: 'metro' })

    expect(result.items.map((a) => a.id)).toEqual(['an1'])
  })

  it('filters by status', async () => {
    apiFetch.mockResolvedValue({
      access_networks: [accessNetworkDto({ id: 'an1', status: 'Active' }), accessNetworkDto({ id: 'an2', status: 'Inactive' })],
    })

    const result = await listAccessNetworks({ status: 'Inactive' })

    expect(result.items.map((a) => a.id)).toEqual(['an2'])
  })

  it('sorts by name ascending by default', async () => {
    apiFetch.mockResolvedValue({
      access_networks: [accessNetworkDto({ id: 'an1', name: 'Zeta' }), accessNetworkDto({ id: 'an2', name: 'Alpha' })],
    })

    const result = await listAccessNetworks()

    expect(result.items.map((a) => a.name)).toEqual(['Alpha', 'Zeta'])
  })

  it('reverses sort direction on request', async () => {
    apiFetch.mockResolvedValue({
      access_networks: [accessNetworkDto({ id: 'an1', name: 'Alpha' }), accessNetworkDto({ id: 'an2', name: 'Zeta' })],
    })

    const result = await listAccessNetworks({ sortDirection: 'desc' })

    expect(result.items.map((a) => a.name)).toEqual(['Zeta', 'Alpha'])
  })

  it('paginates results while reporting the true total', async () => {
    const accessNetworks = Array.from({ length: 20 }, (_, i) => accessNetworkDto({ id: `an${i}`, name: `Network ${i}` }))
    apiFetch.mockResolvedValue({ access_networks: accessNetworks })

    const result = await listAccessNetworks({ page: 2, pageSize: 15 })

    expect(result.total).toBe(20)
    expect(result.items).toHaveLength(5)
  })
})

describe('getAccessNetworkById', () => {
  it('returns the access network when found', async () => {
    apiFetch.mockResolvedValue(accessNetworkDto({ id: 'an1' }))

    const result = await getAccessNetworkById('an1')

    expect(result?.id).toBe('an1')
  })

  it('returns null instead of throwing when the access network does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getAccessNetworkById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getAccessNetworkById('an1')).rejects.toThrow('boom')
  })
})

describe('createAccessNetwork', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(accessNetworkDto({ id: 'new' }))

    await createAccessNetwork({ name: 'Metro North', status: 'Active', description: 'North metro fiber ring' })

    expect(apiFetch).toHaveBeenCalledWith('/access-networks/', {
      method: 'POST',
      body: { name: 'Metro North', status: 'Active', description: 'North metro fiber ring' },
    })
  })
})

describe('updateAccessNetwork', () => {
  it('sends a PUT with the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(accessNetworkDto({ id: 'an1', name: 'Metro North Renamed' }))

    await updateAccessNetwork('an1', { name: 'Metro North Renamed', status: 'Inactive', description: 'Updated' })

    expect(apiFetch).toHaveBeenCalledWith('/access-networks/an1', {
      method: 'PUT',
      body: { name: 'Metro North Renamed', status: 'Inactive', description: 'Updated' },
    })
  })
})

describe('deleteAccessNetwork', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteAccessNetwork('an1')

    expect(apiFetch).toHaveBeenCalledWith('/access-networks/an1', { method: 'DELETE' })
  })
})
