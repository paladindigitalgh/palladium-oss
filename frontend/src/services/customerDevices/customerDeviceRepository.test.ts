import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listCustomerDevices,
  listCustomerDevicesByCustomerId,
  listActiveCustomerDevicesByCustomerId,
  attachCustomerDevice,
  detachCustomerDevice,
  setCustomerDeviceLocation,
} from './customerDeviceRepository'

/** Mirrors serviceEquipmentRepository.test.ts's own doc comment: every list below fetches the same full /customer-devices/ list and filters client-side. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

function customerDeviceDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'cd1',
    customer_id: 'c1',
    device_id: 'd1',
    location_id: null,
    attached_at: '2026-01-01T00:00:00Z',
    detached_at: null,
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listCustomerDevices', () => {
  it('maps every record from the DTO, unfiltered', async () => {
    apiFetch.mockResolvedValue({
      customer_devices: [customerDeviceDto({ id: 'cd1' }), customerDeviceDto({ id: 'cd2' })],
    })

    const result = await listCustomerDevices()

    expect(result.map((item) => item.id)).toEqual(['cd1', 'cd2'])
  })
})

describe('listCustomerDevicesByCustomerId', () => {
  it('returns only records for the given customer', async () => {
    apiFetch.mockResolvedValue({
      customer_devices: [
        customerDeviceDto({ id: 'cd1', customer_id: 'c1' }),
        customerDeviceDto({ id: 'cd2', customer_id: 'c2' }),
      ],
    })

    const result = await listCustomerDevicesByCustomerId('c1')

    expect(result.map((item) => item.id)).toEqual(['cd1'])
  })
})

describe('listActiveCustomerDevicesByCustomerId', () => {
  it('excludes detached records for the given customer', async () => {
    apiFetch.mockResolvedValue({
      customer_devices: [
        customerDeviceDto({ id: 'cd1', customer_id: 'c1', detached_at: null }),
        customerDeviceDto({ id: 'cd2', customer_id: 'c1', detached_at: '2026-02-01T00:00:00Z' }),
      ],
    })

    const result = await listActiveCustomerDevicesByCustomerId('c1')

    expect(result.map((item) => item.id)).toEqual(['cd1'])
  })
})

describe('attachCustomerDevice', () => {
  it('sends the request body in the API wire shape, with attachedAt set to now and detachedAt null', async () => {
    apiFetch.mockResolvedValue(customerDeviceDto({ id: 'cd1', customer_id: 'c1', device_id: 'd1' }))

    const result = await attachCustomerDevice({
      customerId: 'c1',
      deviceId: 'd1',
      locationId: null,
    })

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [path, options] = apiFetch.mock.calls[0]
    expect(path).toBe('/customer-devices/')
    expect(options.method).toBe('POST')
    expect(options.body).toMatchObject({
      customer_id: 'c1',
      device_id: 'd1',
      location_id: null,
      detached_at: null,
    })
    expect(typeof options.body.attached_at).toBe('string')
    expect(result.id).toBe('cd1')
  })

  it('sends a chosen locationId through unchanged', async () => {
    apiFetch.mockResolvedValue(customerDeviceDto({ location_id: 'l1' }))

    await attachCustomerDevice({ customerId: 'c1', deviceId: 'd1', locationId: 'l1' })

    const [, options] = apiFetch.mock.calls[0]
    expect(options.body).toMatchObject({ location_id: 'l1' })
  })
})

describe('detachCustomerDevice', () => {
  it('PUTs the record back with detachedAt set to now and every other field echoed unchanged', async () => {
    const record = {
      id: 'cd1',
      customerId: 'c1',
      deviceId: 'd1',
      locationId: 'l1',
      attachedAt: '2026-01-01T00:00:00Z',
      detachedAt: null,
    }
    apiFetch.mockResolvedValue(customerDeviceDto({ detached_at: '2026-03-01T00:00:00Z' }))

    const result = await detachCustomerDevice(record)

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [path, options] = apiFetch.mock.calls[0]
    expect(path).toBe('/customer-devices/cd1')
    expect(options.method).toBe('PUT')
    expect(options.body).toMatchObject({
      customer_id: 'c1',
      device_id: 'd1',
      location_id: 'l1',
      attached_at: '2026-01-01T00:00:00Z',
    })
    expect(typeof options.body.detached_at).toBe('string')
    expect(result.detachedAt).toBe('2026-03-01T00:00:00Z')
  })
})

describe('setCustomerDeviceLocation', () => {
  it('PUTs the record back with the new locationId and every other field echoed unchanged', async () => {
    const record = {
      id: 'cd1',
      customerId: 'c1',
      deviceId: 'd1',
      locationId: null,
      attachedAt: '2026-01-01T00:00:00Z',
      detachedAt: null,
    }
    apiFetch.mockResolvedValue(customerDeviceDto({ location_id: 'l2' }))

    const result = await setCustomerDeviceLocation(record, 'l2')

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [path, options] = apiFetch.mock.calls[0]
    expect(path).toBe('/customer-devices/cd1')
    expect(options.method).toBe('PUT')
    expect(options.body).toMatchObject({
      customer_id: 'c1',
      device_id: 'd1',
      location_id: 'l2',
      attached_at: '2026-01-01T00:00:00Z',
      detached_at: null,
    })
    expect(result.locationId).toBe('l2')
  })

  it('sends null to clear a previously-set location', async () => {
    const record = {
      id: 'cd1',
      customerId: 'c1',
      deviceId: 'd1',
      locationId: 'l1',
      attachedAt: '2026-01-01T00:00:00Z',
      detachedAt: null,
    }
    apiFetch.mockResolvedValue(customerDeviceDto({ location_id: null }))

    await setCustomerDeviceLocation(record, null)

    const [, options] = apiFetch.mock.calls[0]
    expect(options.body).toMatchObject({ location_id: null })
  })
})
