import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listOLTs, listOLTsByAccessNetworkId, getOLTById, createOLT, updateOLT, deleteOLT } from './oltRepository'

/**
 * Like locationRepository.test.ts, this has no client-side search/sort/
 * pagination -- just list/listByParent. Unlike Location though,
 * getOLTById hits GET /olts/:id directly (OLT has its own Detail page),
 * so it DOES have an ApiError not_found branch to test, same as
 * customerRepository.test.ts's getCustomerById.
 */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function oltDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'olt1',
    access_network_id: 'an1',
    name: 'OLT-Core-1',
    vendor: 'Nokia',
    model: '7360 ISAM',
    management_ip_address: '10.0.0.1',
    connection_profile_id: null,
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listOLTs', () => {
  it('maps every OLT from the DTO', async () => {
    apiFetch.mockResolvedValue({ olts: [oltDto({ id: 'olt1' }), oltDto({ id: 'olt2' })] })

    const result = await listOLTs()

    expect(result.map((o) => o.id)).toEqual(['olt1', 'olt2'])
  })
})

describe('listOLTsByAccessNetworkId', () => {
  it('returns only OLTs belonging to the given access network', async () => {
    apiFetch.mockResolvedValue({
      olts: [
        oltDto({ id: 'olt1', access_network_id: 'an1' }),
        oltDto({ id: 'olt2', access_network_id: 'an2' }),
        oltDto({ id: 'olt3', access_network_id: 'an1' }),
      ],
    })

    const result = await listOLTsByAccessNetworkId('an1')

    expect(result.map((o) => o.id)).toEqual(['olt1', 'olt3'])
  })
})

describe('getOLTById', () => {
  it('returns the OLT when found', async () => {
    apiFetch.mockResolvedValue(oltDto({ id: 'olt1' }))

    const result = await getOLTById('olt1')

    expect(result?.id).toBe('olt1')
  })

  it('returns null instead of throwing when the OLT does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getOLTById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getOLTById('olt1')).rejects.toThrow('boom')
  })
})

describe('createOLT', () => {
  it('sends the request body in the API wire shape, always with a null connection_profile_id', async () => {
    apiFetch.mockResolvedValue(oltDto({ id: 'new' }))

    await createOLT({
      accessNetworkId: 'an1',
      name: 'OLT-Core-1',
      vendor: 'Nokia',
      model: '7360 ISAM',
      managementIpAddress: '10.0.0.1',
      description: 'Core site OLT',
    })

    expect(apiFetch).toHaveBeenCalledWith('/olts/', {
      method: 'POST',
      body: {
        access_network_id: 'an1',
        name: 'OLT-Core-1',
        vendor: 'Nokia',
        model: '7360 ISAM',
        management_ip_address: '10.0.0.1',
        description: 'Core site OLT',
        connection_profile_id: null,
      },
    })
  })
})

describe('updateOLT', () => {
  it('sends a PUT with the request body in the API wire shape, passing through accessNetworkId/connectionProfileId unchanged', async () => {
    apiFetch.mockResolvedValue(oltDto({ id: 'olt1', name: 'OLT-Core-1 Renamed' }))

    await updateOLT('olt1', {
      name: 'OLT-Core-1 Renamed',
      vendor: 'Calix',
      model: 'E7-2',
      managementIpAddress: '10.0.0.2',
      description: 'Updated',
      accessNetworkId: 'an1',
      connectionProfileId: 'cp1',
    })

    expect(apiFetch).toHaveBeenCalledWith('/olts/olt1', {
      method: 'PUT',
      body: {
        access_network_id: 'an1',
        name: 'OLT-Core-1 Renamed',
        vendor: 'Calix',
        model: 'E7-2',
        management_ip_address: '10.0.0.2',
        description: 'Updated',
        connection_profile_id: 'cp1',
      },
    })
  })
})

describe('deleteOLT', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteOLT('olt1')

    expect(apiFetch).toHaveBeenCalledWith('/olts/olt1', { method: 'DELETE' })
  })
})
