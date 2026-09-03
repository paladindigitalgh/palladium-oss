import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listServiceEquipment, listServiceEquipmentByServiceId, listServiceEquipmentByDeviceId } from './serviceEquipmentRepository'

/** Both functions fetch the same full /service-equipment/ list and filter client-side by a different field -- see this file's own doc comment. */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

function equipmentDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'se1',
    service_id: 's1',
    device_id: 'd1',
    role: 'ONU',
    description: '',
    installed_at: '2026-01-01T00:00:00Z',
    removed_at: null,
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listServiceEquipment', () => {
  it('maps every piece of equipment from the DTO, unfiltered', async () => {
    apiFetch.mockResolvedValue({
      service_equipment: [equipmentDto({ id: 'se1', service_id: 's1' }), equipmentDto({ id: 'se2', service_id: 's2' })],
    })

    const result = await listServiceEquipment()

    expect(result.map((item) => item.id)).toEqual(['se1', 'se2'])
  })
})

describe('listServiceEquipmentByServiceId', () => {
  it('maps the DTO and returns only equipment for the given service', async () => {
    apiFetch.mockResolvedValue({
      service_equipment: [
        equipmentDto({ id: 'se1', service_id: 's1', device_id: 'd1' }),
        equipmentDto({ id: 'se2', service_id: 's2', device_id: 'd2' }),
      ],
    })

    const result = await listServiceEquipmentByServiceId('s1')

    expect(result).toEqual([{ id: 'se1', serviceId: 's1', deviceId: 'd1', role: 'ONU', description: '', installedAt: '2026-01-01T00:00:00Z', removedAt: null }])
  })
})

describe('listServiceEquipmentByDeviceId', () => {
  it('returns only equipment for the given device', async () => {
    apiFetch.mockResolvedValue({
      service_equipment: [
        equipmentDto({ id: 'se1', service_id: 's1', device_id: 'd1' }),
        equipmentDto({ id: 'se2', service_id: 's2', device_id: 'd2' }),
      ],
    })

    const result = await listServiceEquipmentByDeviceId('d2')

    expect(result.map((item) => item.id)).toEqual(['se2'])
  })

  it('passes installed_at/removed_at through as-is, including null', async () => {
    apiFetch.mockResolvedValue({
      service_equipment: [equipmentDto({ id: 'se1', device_id: 'd1', installed_at: null, removed_at: null })],
    })

    const result = await listServiceEquipmentByDeviceId('d1')

    expect(result[0].installedAt).toBeNull()
    expect(result[0].removedAt).toBeNull()
  })
})
