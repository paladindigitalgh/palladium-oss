import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import {
  listAccessInterfaces,
  listAccessInterfacesByPONPortId,
  getAccessInterfaceById,
  createAccessInterface,
  deleteAccessInterface,
} from './accessInterfaceRepository'

/** Mirrors oltRepository.test.ts's shape exactly -- list/listByParent/get-by-id-direct/create/delete, no client-side sort or pagination. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function accessInterfaceDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'ai1',
    pon_port_id: 'pp1',
    technology: 'GPON',
    name: 'AI-1',
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

describe('listAccessInterfaces', () => {
  it('maps every access interface from the DTO', async () => {
    apiFetch.mockResolvedValue({ access_interfaces: [accessInterfaceDto({ id: 'ai1' }), accessInterfaceDto({ id: 'ai2' })] })

    const result = await listAccessInterfaces()

    expect(result.map((a) => a.id)).toEqual(['ai1', 'ai2'])
  })
})

describe('listAccessInterfacesByPONPortId', () => {
  it('returns only access interfaces belonging to the given PON port', async () => {
    apiFetch.mockResolvedValue({
      access_interfaces: [
        accessInterfaceDto({ id: 'ai1', pon_port_id: 'pp1' }),
        accessInterfaceDto({ id: 'ai2', pon_port_id: 'pp2' }),
        accessInterfaceDto({ id: 'ai3', pon_port_id: 'pp1' }),
      ],
    })

    const result = await listAccessInterfacesByPONPortId('pp1')

    expect(result.map((a) => a.id)).toEqual(['ai1', 'ai3'])
  })
})

describe('getAccessInterfaceById', () => {
  it('returns the access interface when found', async () => {
    apiFetch.mockResolvedValue(accessInterfaceDto({ id: 'ai1' }))

    const result = await getAccessInterfaceById('ai1')

    expect(result?.id).toBe('ai1')
  })

  it('returns null instead of throwing when the access interface does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getAccessInterfaceById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getAccessInterfaceById('ai1')).rejects.toThrow('boom')
  })
})

describe('createAccessInterface', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(accessInterfaceDto({ id: 'new' }))

    await createAccessInterface({
      ponPortId: 'pp1',
      technology: 'GPON',
      name: 'AI-1',
      status: 'Active',
      description: 'Primary GPON interface',
    })

    expect(apiFetch).toHaveBeenCalledWith('/access-interfaces/', {
      method: 'POST',
      body: { pon_port_id: 'pp1', technology: 'GPON', name: 'AI-1', status: 'Active', description: 'Primary GPON interface' },
    })
  })
})

describe('deleteAccessInterface', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteAccessInterface('ai1')

    expect(apiFetch).toHaveBeenCalledWith('/access-interfaces/ai1', { method: 'DELETE' })
  })
})
