import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listDeviceManufacturers, createDeviceManufacturer, deleteDeviceManufacturer } from './deviceManufacturerRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function deviceManufacturerDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'mfr1',
    name: 'Nokia',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listDeviceManufacturers', () => {
  it('maps every DeviceManufacturer from the DTO', async () => {
    apiFetch.mockResolvedValue({
      device_manufacturers: [deviceManufacturerDto({ id: 'mfr1' }), deviceManufacturerDto({ id: 'mfr2', name: 'Calix' })],
    })

    const result = await listDeviceManufacturers()

    expect(result.map((m) => m.id)).toEqual(['mfr1', 'mfr2'])
    expect(result[1].name).toBe('Calix')
  })
})

describe('createDeviceManufacturer', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(deviceManufacturerDto({ id: 'new' }))

    await createDeviceManufacturer({ name: 'Nokia', description: 'Fiber ONT vendor' })

    expect(apiFetch).toHaveBeenCalledWith('/device-manufacturers/', {
      method: 'POST',
      body: { name: 'Nokia', description: 'Fiber ONT vendor' },
    })
  })
})

describe('deleteDeviceManufacturer', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteDeviceManufacturer('mfr1')

    expect(apiFetch).toHaveBeenCalledWith('/device-manufacturers/mfr1', { method: 'DELETE' })
  })

  it('propagates a conflict error when a DeviceModel still references it', async () => {
    apiFetch.mockRejectedValue(new ApiError('violates a foreign key relationship', 'conflict', 409))

    await expect(deleteDeviceManufacturer('mfr1')).rejects.toThrow('violates a foreign key relationship')
  })
})
