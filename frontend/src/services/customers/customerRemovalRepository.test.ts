import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getCustomerRemovalPreview, executeCustomerRemoval } from './customerRemovalRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

beforeEach(() => {
  apiFetch.mockReset()
})

describe('getCustomerRemovalPreview', () => {
  it('camelCases the full nested preview shape', async () => {
    apiFetch.mockResolvedValue({
      customer_id: 'c1',
      locations: [
        {
          location_id: 'l1',
          name: 'Main St',
          status: 'Active',
          services: [
            {
              service_id: 's1',
              description: 'Residential 500/500',
              status: 'Active',
              equipment: [
                { service_equipment_id: 'se1', device_id: 'd1', device_name: 'ONT-1', role: 'ONU', will_run_olt_teardown: true },
              ],
            },
          ],
        },
      ],
    })

    const result = await getCustomerRemovalPreview('c1')

    expect(apiFetch).toHaveBeenCalledWith('/customers/c1/removal-preview')
    expect(result).toEqual({
      customerId: 'c1',
      locations: [
        {
          locationId: 'l1',
          name: 'Main St',
          status: 'Active',
          services: [
            {
              serviceId: 's1',
              description: 'Residential 500/500',
              status: 'Active',
              equipment: [{ serviceEquipmentId: 'se1', deviceId: 'd1', deviceName: 'ONT-1', role: 'ONU', willRunOltTeardown: true }],
            },
          ],
        },
      ],
    })
  })

  it('returns an empty locations array, not an error, for a customer with nothing attached', async () => {
    apiFetch.mockResolvedValue({ customer_id: 'c1', locations: [] })

    const result = await getCustomerRemovalPreview('c1')

    expect(result).toEqual({ customerId: 'c1', locations: [] })
  })
})

describe('executeCustomerRemoval', () => {
  it('posts to the removal endpoint', async () => {
    apiFetch.mockResolvedValue(undefined)

    await executeCustomerRemoval('c1')

    expect(apiFetch).toHaveBeenCalledWith('/customers/c1/removal', { method: 'POST' })
  })
})
