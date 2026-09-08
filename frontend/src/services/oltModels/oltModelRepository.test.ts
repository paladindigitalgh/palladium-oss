import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listOLTModels, getOLTModelById, createOLTModel, deleteOLTModel } from './oltModelRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function oltModelDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'model1',
    vendor: 'Kontron',
    name: 'C16',
    pon_port_count: 16,
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listOLTModels', () => {
  it('maps every OLTModel from the DTO', async () => {
    apiFetch.mockResolvedValue({ olt_models: [oltModelDto({ id: 'model1' }), oltModelDto({ id: 'model2' })] })

    const result = await listOLTModels()

    expect(result.map((m) => m.id)).toEqual(['model1', 'model2'])
    expect(result[0].ponPortCount).toBe(16)
  })
})

describe('getOLTModelById', () => {
  it('returns the OLTModel when found', async () => {
    apiFetch.mockResolvedValue(oltModelDto({ id: 'model1' }))

    const result = await getOLTModelById('model1')

    expect(result?.id).toBe('model1')
  })

  it('returns null instead of throwing when the OLTModel does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getOLTModelById('missing')

    expect(result).toBeNull()
  })
})

describe('createOLTModel', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(oltModelDto({ id: 'new' }))

    await createOLTModel({ vendor: 'Kontron', name: 'C16', ponPortCount: 16, description: 'Chassis' })

    expect(apiFetch).toHaveBeenCalledWith('/olt-models/', {
      method: 'POST',
      body: { vendor: 'Kontron', name: 'C16', pon_port_count: 16, description: 'Chassis' },
    })
  })
})

describe('deleteOLTModel', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteOLTModel('model1')

    expect(apiFetch).toHaveBeenCalledWith('/olt-models/model1', { method: 'DELETE' })
  })

  it('propagates a conflict error when an OLT still references the model', async () => {
    apiFetch.mockRejectedValue(new ApiError('violates a foreign key relationship', 'conflict', 409))

    await expect(deleteOLTModel('model1')).rejects.toThrow('violates a foreign key relationship')
  })
})
