import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listCustomerEquipmentLocations,
  runONUSummary,
  runONUStatusSummary,
  runONURunningConfig,
  runONUStatus,
  runONUEthernetPorts,
  runDHCPSnoopingEntries,
  runMACAddressTableEntries,
} from './diagnosticsRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listCustomerEquipmentLocations', () => {
  it('maps every location from the DTO, camelCasing field names', async () => {
    apiFetch.mockResolvedValue({
      locations: [
        { service_equipment_id: 'se1', olt_id: 'olt1', interface: 'xgs/1/1' },
        { service_equipment_id: 'se2', olt_id: 'olt2', interface: 'xgs/1/2' },
      ],
    })

    const result = await listCustomerEquipmentLocations('customer1')

    expect(apiFetch).toHaveBeenCalledWith('/diagnostics/customers/customer1/equipment-locations')
    expect(result).toEqual([
      { serviceEquipmentId: 'se1', oltId: 'olt1', interface: 'xgs/1/1' },
      { serviceEquipmentId: 'se2', oltId: 'olt2', interface: 'xgs/1/2' },
    ])
  })

  it('returns an empty array, not an error, when nothing is attached', async () => {
    apiFetch.mockResolvedValue({ locations: [] })

    const result = await listCustomerEquipmentLocations('customer1')

    expect(result).toEqual([])
  })
})

describe('whole-OLT functions', () => {
  const cases: { name: string; fn: (oltId: string) => Promise<string>; path: string }[] = [
    { name: 'runONUSummary', fn: runONUSummary, path: 'onu-summary' },
    { name: 'runONUStatusSummary', fn: runONUStatusSummary, path: 'onu-status-summary' },
  ]

  for (const { name, fn, path } of cases) {
    it(`${name} posts to /diagnostics/olts/:oltId/${path} with no request body and returns the raw output`, async () => {
      apiFetch.mockResolvedValue({ output: 'device output for ' + path })

      const result = await fn('olt1')

      expect(apiFetch).toHaveBeenCalledWith(`/diagnostics/olts/olt1/${path}`, { method: 'POST' })
      expect(result).toBe('device output for ' + path)
    })
  }
})

describe('per-command functions', () => {
  const cases: {
    name: string
    fn: (oltId: string, iface: string) => Promise<string>
    path: string
  }[] = [
    { name: 'runONURunningConfig', fn: runONURunningConfig, path: 'onu-running-config' },
    { name: 'runONUStatus', fn: runONUStatus, path: 'onu-status' },
    { name: 'runONUEthernetPorts', fn: runONUEthernetPorts, path: 'onu-ethernet-ports' },
    { name: 'runDHCPSnoopingEntries', fn: runDHCPSnoopingEntries, path: 'dhcp-snooping-entries' },
    { name: 'runMACAddressTableEntries', fn: runMACAddressTableEntries, path: 'mac-address-table-entries' },
  ]

  for (const { name, fn, path } of cases) {
    it(`${name} posts to /diagnostics/olts/:oltId/${path} with the interface in the body and returns the raw output`, async () => {
      apiFetch.mockResolvedValue({ output: 'device output for ' + path })

      const result = await fn('olt1', 'xgs/1/1')

      expect(apiFetch).toHaveBeenCalledWith(`/diagnostics/olts/olt1/${path}`, {
        method: 'POST',
        body: { interface: 'xgs/1/1' },
      })
      expect(result).toBe('device output for ' + path)
    })
  }
})
