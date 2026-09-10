import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listDeviceModels, createDeviceModel, deleteDeviceModel, setDeviceModelDefault } from './deviceModelRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function deviceModelDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'model1',
    manufacturer_id: 'mfr1',
    name: 'G-140W-CT',
    description: '',
    is_default: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listDeviceModels', () => {
  it('maps every DeviceModel from the DTO', async () => {
    apiFetch.mockResolvedValue({ device_models: [deviceModelDto({ id: 'model1' }), deviceModelDto({ id: 'model2' })] })

    const result = await listDeviceModels()

    expect(result.map((m) => m.id)).toEqual(['model1', 'model2'])
    expect(result[0].manufacturerId).toBe('mfr1')
  })
})

describe('createDeviceModel', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(deviceModelDto({ id: 'new' }))

    await createDeviceModel({ manufacturerId: 'mfr1', name: 'G-140W-CT', description: 'GPON ONT' })

    expect(apiFetch).toHaveBeenCalledWith('/device-models/', {
      method: 'POST',
      body: { manufacturer_id: 'mfr1', name: 'G-140W-CT', description: 'GPON ONT' },
    })
  })
})

describe('deleteDeviceModel', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteDeviceModel('model1')

    expect(apiFetch).toHaveBeenCalledWith('/device-models/model1', { method: 'DELETE' })
  })

  it('propagates a conflict error when a Device still references it', async () => {
    apiFetch.mockRejectedValue(new ApiError('violates a foreign key relationship', 'conflict', 409))

    await expect(deleteDeviceModel('model1')).rejects.toThrow('violates a foreign key relationship')
  })
})

describe('setDeviceModelDefault', () => {
  it('PUTs to the dedicated /default endpoint and maps the returned DTO', async () => {
    apiFetch.mockResolvedValue(deviceModelDto({ id: 'model1', is_default: true }))

    const result = await setDeviceModelDefault('model1', true)

    expect(apiFetch).toHaveBeenCalledWith('/device-models/model1/default', {
      method: 'PUT',
      body: { is_default: true },
    })
    expect(result.isDefault).toBe(true)
  })

  it('sends false to clear the default', async () => {
    apiFetch.mockResolvedValue(deviceModelDto({ id: 'model1', is_default: false }))

    await setDeviceModelDefault('model1', false)

    expect(apiFetch).toHaveBeenCalledWith('/device-models/model1/default', {
      method: 'PUT',
      body: { is_default: false },
    })
  })
})
