import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listOLTs, getOLTById, createOLT, updateOLT, deleteOLT } from './oltRepository'

/**
 * Like locationRepository.test.ts, this has no client-side search/sort/
 * pagination -- just list. Unlike Location though, getOLTById hits GET
 * /olts/:id directly (OLT has its own Detail page), so it DOES have an
 * ApiError not_found branch to test, same as customerRepository.test.ts's
 * getCustomerById.
 */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function oltDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'olt1',
    name: 'OLT-Core-1',
    olt_model_id: 'model1',
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
  it('sends the request body in the API wire shape, including a null connection_profile_id when none is chosen', async () => {
    apiFetch.mockResolvedValue(oltDto({ id: 'new' }))

    await createOLT({
      name: 'OLT-Core-1',
      oltModelId: 'model1',
      managementIpAddress: '10.0.0.1',
      description: 'Core site OLT',
      connectionProfileId: null,
    })

    expect(apiFetch).toHaveBeenCalledWith('/olts/', {
      method: 'POST',
      body: {
        name: 'OLT-Core-1',
        olt_model_id: 'model1',
        management_ip_address: '10.0.0.1',
        description: 'Core site OLT',
        connection_profile_id: null,
      },
    })
  })

  it('sends the chosen connection_profile_id when one is set', async () => {
    apiFetch.mockResolvedValue(oltDto({ id: 'new' }))

    await createOLT({
      name: 'OLT-Core-1',
      oltModelId: 'model1',
      managementIpAddress: '10.0.0.1',
      description: 'Core site OLT',
      connectionProfileId: 'cp1',
    })

    expect(apiFetch).toHaveBeenCalledWith(
      '/olts/',
      expect.objectContaining({ body: expect.objectContaining({ connection_profile_id: 'cp1' }) }),
    )
  })
})

describe('updateOLT', () => {
  it('sends a PUT with the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(oltDto({ id: 'olt1', name: 'OLT-Core-1 Renamed' }))

    await updateOLT('olt1', {
      name: 'OLT-Core-1 Renamed',
      oltModelId: 'model2',
      managementIpAddress: '10.0.0.2',
      description: 'Updated',
      connectionProfileId: 'cp1',
    })

    expect(apiFetch).toHaveBeenCalledWith('/olts/olt1', {
      method: 'PUT',
      body: {
        name: 'OLT-Core-1 Renamed',
        olt_model_id: 'model2',
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
