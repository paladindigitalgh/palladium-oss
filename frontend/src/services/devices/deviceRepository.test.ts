import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listDevices, listDevicesByRackId, getDeviceById, createDevice, updateDevice, deleteDevice } from './deviceRepository'

/** Mirrors customerRepository.test.ts's shape exactly -- see that file. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function deviceDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'd1',
    name: 'ONT-1',
    description: 'Lobby ONT',
    rack_id: null,
    manufacturer: 'Nokia',
    model: 'G-010G',
    serial_number: 'SN123',
    asset_tag: 'AT-1',
    status: 'Installed',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listDevices', () => {
  it('filters by search term across name, serial number, manufacturer, and model', async () => {
    apiFetch.mockResolvedValue({
      devices: [
        deviceDto({ id: 'd1', name: 'ONT-1', serial_number: 'AAA', manufacturer: 'Nokia', model: 'G-010G' }),
        deviceDto({ id: 'd2', name: 'Switch-1', serial_number: 'BBB', manufacturer: 'Cisco', model: 'X-100' }),
      ],
    })

    expect((await listDevices({ search: 'ont-1' })).items.map((d) => d.id)).toEqual(['d1'])
    expect((await listDevices({ search: 'BBB' })).items.map((d) => d.id)).toEqual(['d2'])
    expect((await listDevices({ search: 'cisco' })).items.map((d) => d.id)).toEqual(['d2'])
    expect((await listDevices({ search: 'x-100' })).items.map((d) => d.id)).toEqual(['d2'])
  })

  it('filters by status', async () => {
    apiFetch.mockResolvedValue({
      devices: [deviceDto({ id: 'd1', status: 'Installed' }), deviceDto({ id: 'd2', status: 'Retired' })],
    })

    const result = await listDevices({ status: 'Retired' })

    expect(result.items.map((d) => d.id)).toEqual(['d2'])
  })

  it('sorts by name ascending by default', async () => {
    apiFetch.mockResolvedValue({ devices: [deviceDto({ id: 'd1', name: 'Zeta' }), deviceDto({ id: 'd2', name: 'Alpha' })] })

    const result = await listDevices()

    expect(result.items.map((d) => d.name)).toEqual(['Alpha', 'Zeta'])
  })

  it('sorts by status when requested', async () => {
    apiFetch.mockResolvedValue({
      devices: [deviceDto({ id: 'd1', status: 'Retired' }), deviceDto({ id: 'd2', status: 'InStock' })],
    })

    const result = await listDevices({ sortKey: 'status' })

    expect(result.items.map((d) => d.status)).toEqual(['InStock', 'Retired'])
  })

  it('paginates results while reporting the true total', async () => {
    const devices = Array.from({ length: 20 }, (_, i) => deviceDto({ id: `d${i}`, name: `Device ${i}` }))
    apiFetch.mockResolvedValue({ devices })

    const result = await listDevices({ page: 2, pageSize: 15 })

    expect(result.total).toBe(20)
    expect(result.items).toHaveLength(5)
  })
})

describe('listDevicesByRackId', () => {
  it('returns only devices racked in the given rack', async () => {
    apiFetch.mockResolvedValue({
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
  it('returns the device when found', async () => {
    apiFetch.mockResolvedValue(deviceDto({ id: 'd1' }))

    const result = await getDeviceById('d1')

    expect(result?.id).toBe('d1')
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

describe('createDevice', () => {
  it('sends the request body in the API wire shape, with a null rack_id when no rack is chosen', async () => {
    apiFetch.mockResolvedValue(deviceDto({ id: 'new' }))

    await createDevice({
      name: 'ONT-2',
      manufacturer: 'Nokia',
      model: 'G-010G',
      serialNumber: 'SN999',
      assetTag: 'AT-9',
      status: 'InStock',
      description: 'Spare',
      rackId: null,
    })

    expect(apiFetch).toHaveBeenCalledWith('/devices/', {
      method: 'POST',
      body: {
        name: 'ONT-2',
        manufacturer: 'Nokia',
        model: 'G-010G',
        serial_number: 'SN999',
        asset_tag: 'AT-9',
        status: 'InStock',
        description: 'Spare',
        rack_id: null,
      },
    })
  })

  it('sends the chosen rack_id when a rack is selected', async () => {
    apiFetch.mockResolvedValue(deviceDto({ id: 'new', rack_id: 'rack-1' }))

    await createDevice({
      name: 'ONT-2',
      manufacturer: 'Nokia',
      model: 'G-010G',
      serialNumber: 'SN999',
      assetTag: 'AT-9',
      status: 'InStock',
      description: 'Spare',
      rackId: 'rack-1',
    })

    const [, init] = apiFetch.mock.calls[0]
    expect((init.body as { rack_id: string | null }).rack_id).toBe('rack-1')
  })
})

describe('updateDevice', () => {
  it('sends the request body as a PUT, passing the given rackId through unchanged', async () => {
    apiFetch.mockResolvedValue(deviceDto({ id: 'd1', rack_id: 'rack-1' }))

    await updateDevice('d1', {
      name: 'ONT-1 Renamed',
      manufacturer: 'Nokia',
      model: 'G-010G',
      serialNumber: 'SN123',
      assetTag: 'AT-1',
      status: 'Maintenance',
      description: 'Lobby ONT',
      rackId: 'rack-1',
    })

    expect(apiFetch).toHaveBeenCalledWith('/devices/d1', {
      method: 'PUT',
      body: {
        name: 'ONT-1 Renamed',
        manufacturer: 'Nokia',
        model: 'G-010G',
        serial_number: 'SN123',
        asset_tag: 'AT-1',
        status: 'Maintenance',
        description: 'Lobby ONT',
        rack_id: 'rack-1',
      },
    })
  })

  it('sends a null rack_id through unchanged when the device was never racked', async () => {
    apiFetch.mockResolvedValue(deviceDto({ id: 'd1' }))

    await updateDevice('d1', {
      name: 'ONT-1',
      manufacturer: 'Nokia',
      model: 'G-010G',
      serialNumber: 'SN123',
      assetTag: 'AT-1',
      status: 'Installed',
      description: '',
      rackId: null,
    })

    const [, init] = apiFetch.mock.calls[0]
    expect((init.body as { rack_id: string | null }).rack_id).toBeNull()
  })
})

describe('deleteDevice', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteDevice('d1')

    expect(apiFetch).toHaveBeenCalledWith('/devices/d1', { method: 'DELETE' })
  })
})
