import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listPONPorts, listPONPortsByOLTId, getPONPortById, createPONPort, deletePONPort } from './ponPortRepository'

/** Mirrors oltRepository.test.ts's shape exactly -- list/listByParent/get-by-id-direct/create/delete, no client-side sort or pagination. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function ponPortDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'pp1',
    olt_id: 'olt1',
    port_number: 1,
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listPONPorts', () => {
  it('maps every PON port from the DTO', async () => {
    apiFetch.mockResolvedValue({ pon_ports: [ponPortDto({ id: 'pp1' }), ponPortDto({ id: 'pp2' })] })

    const result = await listPONPorts()

    expect(result.map((p) => p.id)).toEqual(['pp1', 'pp2'])
  })
})

describe('listPONPortsByOLTId', () => {
  it('returns only PON ports belonging to the given OLT', async () => {
    apiFetch.mockResolvedValue({
      pon_ports: [
        ponPortDto({ id: 'pp1', olt_id: 'olt1' }),
        ponPortDto({ id: 'pp2', olt_id: 'olt2' }),
        ponPortDto({ id: 'pp3', olt_id: 'olt1' }),
      ],
    })

    const result = await listPONPortsByOLTId('olt1')

    expect(result.map((p) => p.id)).toEqual(['pp1', 'pp3'])
  })
})

describe('getPONPortById', () => {
  it('returns the PON port when found', async () => {
    apiFetch.mockResolvedValue(ponPortDto({ id: 'pp1' }))

    const result = await getPONPortById('pp1')

    expect(result?.id).toBe('pp1')
  })

  it('returns null instead of throwing when the PON port does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getPONPortById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getPONPortById('pp1')).rejects.toThrow('boom')
  })
})

describe('createPONPort', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(ponPortDto({ id: 'new' }))

    await createPONPort({ oltId: 'olt1', portNumber: 3, description: 'Rack 2 PON port 3' })

    expect(apiFetch).toHaveBeenCalledWith('/pon-ports/', {
      method: 'POST',
      body: { olt_id: 'olt1', port_number: 3, description: 'Rack 2 PON port 3' },
    })
  })
})

describe('deletePONPort', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deletePONPort('pp1')

    expect(apiFetch).toHaveBeenCalledWith('/pon-ports/pp1', { method: 'DELETE' })
  })
})
