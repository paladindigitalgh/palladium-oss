import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listServices, listServicesByLocationIds, getServiceById, createService, updateService, deleteService } from './serviceRepository'

/** Mirrors customerRepository.test.ts's shape exactly -- see that file. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function serviceDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 's1',
    location_id: 'l1',
    product_id: 'p1',
    service_profile_id: 'sp1',
    status: 'Active',
    description: '',
    activated_at: null,
    suspended_at: null,
    disconnected_at: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listServices', () => {
  it('filters by search term across id and description', async () => {
    apiFetch.mockResolvedValue({
      services: [serviceDto({ id: 's1', description: 'Fiber 1G' }), serviceDto({ id: 's2', description: 'Coax 100M' })],
    })

    expect((await listServices({ search: 's1' })).items.map((s) => s.id)).toEqual(['s1'])
    expect((await listServices({ search: 'coax' })).items.map((s) => s.id)).toEqual(['s2'])
  })

  it('filters by status', async () => {
    apiFetch.mockResolvedValue({
      services: [serviceDto({ id: 's1', status: 'Active' }), serviceDto({ id: 's2', status: 'Suspended' })],
    })

    const result = await listServices({ status: 'Suspended' })

    expect(result.items.map((s) => s.id)).toEqual(['s2'])
  })

  it('sorts by id ascending by default', async () => {
    apiFetch.mockResolvedValue({ services: [serviceDto({ id: 's2' }), serviceDto({ id: 's1' })] })

    const result = await listServices()

    expect(result.items.map((s) => s.id)).toEqual(['s1', 's2'])
  })

  it('sorts by status when requested, reversed on request', async () => {
    apiFetch.mockResolvedValue({
      services: [serviceDto({ id: 's1', status: 'Active' }), serviceDto({ id: 's2', status: 'Suspended' })],
    })

    const result = await listServices({ sortKey: 'status', sortDirection: 'desc' })

    expect(result.items.map((s) => s.status)).toEqual(['Suspended', 'Active'])
  })

  it('paginates results while reporting the true total', async () => {
    const services = Array.from({ length: 20 }, (_, i) => serviceDto({ id: `s${i}` }))
    apiFetch.mockResolvedValue({ services })

    const result = await listServices({ page: 2, pageSize: 15 })

    expect(result.total).toBe(20)
    expect(result.items).toHaveLength(5)
  })
})

describe('listServicesByLocationIds', () => {
  it('returns only services whose locationId is in the given set', async () => {
    apiFetch.mockResolvedValue({
      services: [
        serviceDto({ id: 's1', location_id: 'l1' }),
        serviceDto({ id: 's2', location_id: 'l2' }),
        serviceDto({ id: 's3', location_id: 'l3' }),
      ],
    })

    const result = await listServicesByLocationIds(['l1', 'l3'])

    expect(result.map((s) => s.id)).toEqual(['s1', 's3'])
  })
})

describe('getServiceById', () => {
  it('returns the service when found', async () => {
    apiFetch.mockResolvedValue(serviceDto({ id: 's1' }))

    const result = await getServiceById('s1')

    expect(result?.id).toBe('s1')
  })

  it('returns null instead of throwing when the service does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getServiceById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getServiceById('s1')).rejects.toThrow('boom')
  })
})

describe('createService', () => {
  it('sends the request body in the API wire shape, with lifecycle timestamps always null', async () => {
    apiFetch.mockResolvedValue(serviceDto({ id: 'new' }))

    await createService({
      locationId: 'l1',
      productId: 'p1',
      serviceProfileId: 'sp1',
      status: 'Pending',
      description: 'New service',
    })

    expect(apiFetch).toHaveBeenCalledWith('/services/', {
      method: 'POST',
      body: {
        location_id: 'l1',
        product_id: 'p1',
        service_profile_id: 'sp1',
        status: 'Pending',
        description: 'New service',
        activated_at: null,
        suspended_at: null,
        disconnected_at: null,
      },
    })
  })
})

describe('updateService', () => {
  it('sends a PUT with the request body in the API wire shape, passing through activated/suspended/disconnected unchanged', async () => {
    apiFetch.mockResolvedValue(serviceDto({ id: 's1', status: 'Suspended' }))

    await updateService('s1', {
      locationId: 'l1',
      productId: 'p1',
      serviceProfileId: 'sp1',
      status: 'Suspended',
      description: 'Updated',
      activatedAt: '2026-02-01T00:00:00Z',
      suspendedAt: '2026-03-01T00:00:00Z',
      disconnectedAt: null,
    })

    expect(apiFetch).toHaveBeenCalledWith('/services/s1', {
      method: 'PUT',
      body: {
        location_id: 'l1',
        product_id: 'p1',
        service_profile_id: 'sp1',
        status: 'Suspended',
        description: 'Updated',
        activated_at: '2026-02-01T00:00:00Z',
        suspended_at: '2026-03-01T00:00:00Z',
        disconnected_at: null,
      },
    })
  })
})

describe('deleteService', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteService('s1')

    expect(apiFetch).toHaveBeenCalledWith('/services/s1', { method: 'DELETE' })
  })
})
