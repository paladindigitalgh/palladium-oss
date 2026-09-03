import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listLocations, listLocationsByCustomerId, getLocationById, createLocation, deleteLocation } from './locationRepository'

/**
 * Unlike customer/service/device, this repository has no client-side
 * search/sort/pagination -- just three ways of slicing one fetched list
 * (see this file's own doc comment). getLocationById filters the full
 * list and returns null when absent; it does not go through apiFetch's
 * own not_found catch (there is no per-id GET endpoint here), so there is
 * no ApiError branch to test for it.
 */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function locationDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'l1',
    customer_id: 'c1',
    name: 'Main Office',
    type: 'Office',
    status: 'Active',
    address1: '123 Main St',
    address2: '',
    city: 'Springfield',
    state: 'IL',
    postal_code: '62704',
    country: 'US',
    description: '',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listLocations', () => {
  it('maps every location from the DTO', async () => {
    apiFetch.mockResolvedValue({ locations: [locationDto({ id: 'l1' }), locationDto({ id: 'l2' })] })

    const result = await listLocations()

    expect(result.map((l) => l.id)).toEqual(['l1', 'l2'])
  })
})

describe('listLocationsByCustomerId', () => {
  it('returns only locations belonging to the given customer', async () => {
    apiFetch.mockResolvedValue({
      locations: [
        locationDto({ id: 'l1', customer_id: 'c1' }),
        locationDto({ id: 'l2', customer_id: 'c2' }),
        locationDto({ id: 'l3', customer_id: 'c1' }),
      ],
    })

    const result = await listLocationsByCustomerId('c1')

    expect(result.map((l) => l.id)).toEqual(['l1', 'l3'])
  })
})

describe('getLocationById', () => {
  it('returns the location when found', async () => {
    apiFetch.mockResolvedValue({ locations: [locationDto({ id: 'l1' }), locationDto({ id: 'l2' })] })

    const result = await getLocationById('l2')

    expect(result?.id).toBe('l2')
  })

  it('returns null instead of throwing when the location does not exist', async () => {
    apiFetch.mockResolvedValue({ locations: [locationDto({ id: 'l1' })] })

    const result = await getLocationById('missing')

    expect(result).toBeNull()
  })
})

describe('createLocation', () => {
  it('sends the request body in the API wire shape, always with an empty description', async () => {
    apiFetch.mockResolvedValue(locationDto({ id: 'new' }))

    await createLocation({
      customerId: 'c1',
      name: 'Warehouse',
      type: 'Warehouse',
      status: 'Active',
      address1: '456 Side St',
      address2: 'Suite 2',
      city: 'Springfield',
      state: 'IL',
      postalCode: '62704',
      country: 'US',
    })

    expect(apiFetch).toHaveBeenCalledWith('/locations/', {
      method: 'POST',
      body: {
        customer_id: 'c1',
        name: 'Warehouse',
        type: 'Warehouse',
        status: 'Active',
        address1: '456 Side St',
        address2: 'Suite 2',
        city: 'Springfield',
        state: 'IL',
        postal_code: '62704',
        country: 'US',
        description: '',
      },
    })
  })
})

describe('deleteLocation', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteLocation('l1')

    expect(apiFetch).toHaveBeenCalledWith('/locations/l1', { method: 'DELETE' })
  })
})
