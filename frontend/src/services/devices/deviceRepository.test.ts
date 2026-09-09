import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import {
  listDevices,
  listDevicesByRackId,
  getDeviceById,
  getDeviceBySerialNumber,
  createDevice,
  authorizeAndCreateDevice,
  updateDevice,
} from './deviceRepository'

/** Mirrors customerRepository.test.ts's shape exactly -- see that file. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

const deviceModelsResponse = {
  device_models: [
    {
      id: 'model-nokia-g010g',
      manufacturer_id: 'mfr-nokia',
      name: 'G-010G',
      description: '',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
    {
      id: 'model-cisco-x100',
      manufacturer_id: 'mfr-cisco',
      name: 'X-100',
      description: '',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
  ],
}

const deviceManufacturersResponse = {
  device_manufacturers: [
    { id: 'mfr-nokia', name: 'Nokia', description: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
    { id: 'mfr-cisco', name: 'Cisco', description: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  ],
}

function deviceDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'd1',
    name: 'ONT-1',
    description: 'Lobby ONT',
    rack_id: null,
    device_model_id: 'model-nokia-g010g',
    serial_number: 'SN123',
    asset_tag: 'AT-1',
    status: 'Active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

/**
 * Every deviceRepository.ts function that returns a Device also fetches
 * the Device Model and Device Manufacturer catalogs to resolve the
 * display manufacturer/model strings (see fromDto's own doc comment).
 * This routes the single mocked apiFetch by URL so a test only has to
 * say what /devices/... should return -- the catalog responses are
 * always these two fixtures above, resolving "model-nokia-g010g" to
 * Nokia/G-010G and "model-cisco-x100" to Cisco/X-100.
 */
function mockApiFetch(devicesResult: unknown) {
  apiFetch.mockImplementation(async (url: string) => {
    if (url === '/device-models/') return deviceModelsResponse
    if (url === '/device-manufacturers/') return deviceManufacturersResponse
    return devicesResult
  })
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listDevices', () => {
  it('filters by search term across name, serial number, manufacturer, and model', async () => {
    mockApiFetch({
      devices: [
        deviceDto({ id: 'd1', name: 'ONT-1', serial_number: 'AAA', device_model_id: 'model-nokia-g010g' }),
        deviceDto({ id: 'd2', name: 'Switch-1', serial_number: 'BBB', device_model_id: 'model-cisco-x100' }),
      ],
    })

    expect((await listDevices({ search: 'ont-1' })).items.map((d) => d.id)).toEqual(['d1'])
    expect((await listDevices({ search: 'BBB' })).items.map((d) => d.id)).toEqual(['d2'])
    expect((await listDevices({ search: 'cisco' })).items.map((d) => d.id)).toEqual(['d2'])
    expect((await listDevices({ search: 'x-100' })).items.map((d) => d.id)).toEqual(['d2'])
  })

  it('resolves manufacturer/model display strings by joining the Device Model and Device Manufacturer catalogs', async () => {
    mockApiFetch({ devices: [deviceDto({ id: 'd1', device_model_id: 'model-nokia-g010g' })] })

    const result = await listDevices()

    expect(result.items[0].manufacturer).toBe('Nokia')
    expect(result.items[0].model).toBe('G-010G')
  })

  it('filters by status', async () => {
    mockApiFetch({
      devices: [deviceDto({ id: 'd1', status: 'Active' }), deviceDto({ id: 'd2', status: 'Retired' })],
    })

    const result = await listDevices({ status: 'Retired' })

    expect(result.items.map((d) => d.id)).toEqual(['d2'])
  })

  it('excludes Retired devices from the default (status: all) view', async () => {
    mockApiFetch({
      devices: [deviceDto({ id: 'd1', status: 'Active' }), deviceDto({ id: 'd2', status: 'Retired' })],
    })

    const result = await listDevices()

    expect(result.items.map((d) => d.id)).toEqual(['d1'])
  })

  it('includes Retired devices when includeRetired is set', async () => {
    mockApiFetch({
      devices: [deviceDto({ id: 'd1', status: 'Active' }), deviceDto({ id: 'd2', status: 'Retired' })],
    })

    const result = await listDevices({ includeRetired: true })

    expect(result.items.map((d) => d.id).sort()).toEqual(['d1', 'd2'])
  })

  it('does not apply includeRetired when a specific status is picked', async () => {
    mockApiFetch({
      devices: [deviceDto({ id: 'd1', status: 'Retired' }), deviceDto({ id: 'd2', status: 'Unused' })],
    })

    const result = await listDevices({ status: 'Retired', includeRetired: false })

    expect(result.items.map((d) => d.id)).toEqual(['d1'])
  })

  it('sorts by name ascending by default', async () => {
    mockApiFetch({ devices: [deviceDto({ id: 'd1', name: 'Zeta' }), deviceDto({ id: 'd2', name: 'Alpha' })] })

    const result = await listDevices()

    expect(result.items.map((d) => d.name)).toEqual(['Alpha', 'Zeta'])
  })

  it('sorts by status when requested', async () => {
    mockApiFetch({
      devices: [deviceDto({ id: 'd1', status: 'Retired' }), deviceDto({ id: 'd2', status: 'Active' })],
    })

    const result = await listDevices({ sortKey: 'status', includeRetired: true })

    expect(result.items.map((d) => d.status)).toEqual(['Active', 'Retired'])
  })

  it('paginates results while reporting the true total', async () => {
    const devices = Array.from({ length: 20 }, (_, i) => deviceDto({ id: `d${i}`, name: `Device ${i}` }))
    mockApiFetch({ devices })

    const result = await listDevices({ page: 2, pageSize: 15 })

    expect(result.total).toBe(20)
    expect(result.items).toHaveLength(5)
  })
})

describe('listDevicesByRackId', () => {
  it('returns only devices racked in the given rack', async () => {
    mockApiFetch({
      devices: [
        deviceDto({ id: 'd1', rack_id: 'rack-1' }),
        deviceDto({ id: 'd2', rack_id: 'rack-2' }),
        deviceDto({ id: 'd3', rack_id: null }),
        deviceDto({ id: 'd4', rack_id: 'rack-1' }),
      ],
    })

    const result = await listDevicesByRackId('rack-1')

    expect(result.map((d) => d.id)).toEqual(['d1', 'd4'])
  })
})

describe('getDeviceById', () => {
  it('returns the device when found, with manufacturer/model resolved', async () => {
    mockApiFetch(deviceDto({ id: 'd1' }))

    const result = await getDeviceById('d1')

    expect(result?.id).toBe('d1')
    expect(result?.manufacturer).toBe('Nokia')
    expect(result?.model).toBe('G-010G')
  })

  it('returns null instead of throwing when the device does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getDeviceById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getDeviceById('d1')).rejects.toThrow('boom')
  })
})

describe('getDeviceBySerialNumber', () => {
  it('returns the device when found', async () => {
    mockApiFetch(deviceDto({ id: 'd1', serial_number: 'SN123' }))

    const result = await getDeviceBySerialNumber('SN123')

    expect(apiFetch).toHaveBeenCalledWith('/devices/by-serial-number/SN123')
    expect(result?.serialNumber).toBe('SN123')
  })

  it('returns null instead of throwing when no device has this serial number', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getDeviceBySerialNumber('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getDeviceBySerialNumber('SN123')).rejects.toThrow('boom')
  })
})

describe('createDevice', () => {
  it('sends the request body in the API wire shape, with a null rack_id when no rack is chosen', async () => {
    mockApiFetch(deviceDto({ id: 'new' }))

    await createDevice({
      name: 'ONT-2',
      deviceModelId: 'model-nokia-g010g',
      serialNumber: 'SN999',
      assetTag: 'AT-9',
      status: 'Unused',
      description: 'Spare',
      rackId: null,
    })

    expect(apiFetch).toHaveBeenCalledWith('/devices/', {
      method: 'POST',
      body: {
        name: 'ONT-2',
        device_model_id: 'model-nokia-g010g',
        serial_number: 'SN999',
        asset_tag: 'AT-9',
        status: 'Unused',
        description: 'Spare',
        rack_id: null,
      },
    })
  })

  it('sends the chosen rack_id when a rack is selected', async () => {
    mockApiFetch(deviceDto({ id: 'new', rack_id: 'rack-1' }))

    await createDevice({
      name: 'ONT-2',
      deviceModelId: 'model-nokia-g010g',
      serialNumber: 'SN999',
      assetTag: 'AT-9',
      status: 'Unused',
      description: 'Spare',
      rackId: 'rack-1',
    })

    const [, init] = apiFetch.mock.calls.find(([url]) => url === '/devices/')!
    expect((init.body as { rack_id: string | null }).rack_id).toBe('rack-1')
  })
})

describe('authorizeAndCreateDevice', () => {
  it('posts oltId/port alongside the device fields to the authorize-and-create-device endpoint', async () => {
    mockApiFetch(deviceDto({ id: 'new', serial_number: 'ISKT001', status: 'Unused' }))

    const device = await authorizeAndCreateDevice('olt1', 'xgs/6', {
      name: 'Discovered ONT',
      deviceModelId: 'model-nokia-g010g',
      serialNumber: 'ISKT001',
      assetTag: '',
      status: 'Unused',
      description: '',
      rackId: null,
    })

    expect(apiFetch).toHaveBeenCalledWith('/provisioning/olts/olt1/authorize-and-create-device', {
      method: 'POST',
      body: {
        port: 'xgs/6',
        name: 'Discovered ONT',
        device_model_id: 'model-nokia-g010g',
        serial_number: 'ISKT001',
        asset_tag: '',
        status: 'Unused',
        description: '',
        rack_id: null,
      },
    })
    expect(device.id).toBe('new')
    expect(device.serialNumber).toBe('ISKT001')
    expect(device.manufacturer).toBe('Nokia')
  })
})

describe('updateDevice', () => {
  it('sends the request body as a PUT, passing the given rackId through unchanged', async () => {
    mockApiFetch(deviceDto({ id: 'd1', rack_id: 'rack-1' }))

    await updateDevice('d1', {
      name: 'ONT-1 Renamed',
      deviceModelId: 'model-nokia-g010g',
      serialNumber: 'SN123',
      assetTag: 'AT-1',
      status: 'Active',
      description: 'Lobby ONT',
      rackId: 'rack-1',
    })

    expect(apiFetch).toHaveBeenCalledWith('/devices/d1', {
      method: 'PUT',
      body: {
        name: 'ONT-1 Renamed',
        device_model_id: 'model-nokia-g010g',
        serial_number: 'SN123',
        asset_tag: 'AT-1',
        status: 'Active',
        description: 'Lobby ONT',
        rack_id: 'rack-1',
      },
    })
  })

  it('sends a null rack_id through unchanged when the device was never racked', async () => {
    mockApiFetch(deviceDto({ id: 'd1' }))

    await updateDevice('d1', {
      name: 'ONT-1',
      deviceModelId: 'model-nokia-g010g',
      serialNumber: 'SN123',
      assetTag: 'AT-1',
      status: 'Active',
      description: '',
      rackId: null,
    })

    const [, init] = apiFetch.mock.calls.find(([url]) => url === '/devices/d1')!
    expect((init.body as { rack_id: string | null }).rack_id).toBeNull()
  })
})
