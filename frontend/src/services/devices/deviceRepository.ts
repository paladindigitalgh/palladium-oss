import type { Device, DeviceStatus, DeviceType } from '@/types/device'
import type { ServiceTechnology } from '@/types/customer'
import { DEVICES } from './deviceDataset'

/**
 * The mock Device data source, mirroring
 * services/customers/customerRepository.ts's shape and role: the Device
 * Collection Workspace only ever calls listDevices/getDeviceById/
 * listAvailableLocations, exactly what a real HTTP-backed implementation
 * would expose. Components never import DEVICES or deviceDataset.ts
 * directly.
 */

export interface DeviceListQuery {
  search?: string
  status?: DeviceStatus | 'all'
  type?: DeviceType | 'all'
  technology?: ServiceTechnology | 'any'
  location?: string | 'all'
  sortKey?: 'device' | 'status' | 'location' | 'assignedCustomer'
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface DeviceListResult {
  items: Device[]
  total: number
}

const SIMULATED_LATENCY_MS = 250

function simulateLatency(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, SIMULATED_LATENCY_MS))
}

function matchesSearch(device: Device, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return (
    device.model.toLowerCase().includes(needle) ||
    device.serialNumber.toLowerCase().includes(needle) ||
    (device.assignedCustomerName?.toLowerCase().includes(needle) ?? false) ||
    device.location.toLowerCase().includes(needle)
  )
}

// A technician sorting by Status wants problems grouped together, not an
// alphabetical listing -- worst-first ordering under ascending.
const STATUS_RANK: Record<DeviceStatus, number> = {
  offline: 0,
  warning: 1,
  provisioning: 2,
  online: 3,
}

function compareDevices(
  sortKey: NonNullable<DeviceListQuery['sortKey']>,
  sortDirection: NonNullable<DeviceListQuery['sortDirection']>,
) {
  const direction = sortDirection === 'desc' ? -1 : 1
  return (a: Device, b: Device): number => {
    let comparison = 0
    switch (sortKey) {
      case 'status':
        comparison = STATUS_RANK[a.status] - STATUS_RANK[b.status]
        break
      case 'location':
        comparison = a.location.localeCompare(b.location)
        break
      case 'assignedCustomer':
        comparison = (a.assignedCustomerName ?? '').localeCompare(b.assignedCustomerName ?? '')
        break
      case 'device':
      default:
        comparison = a.model.localeCompare(b.model)
    }
    return comparison * direction
  }
}

/** Simulates a paginated, filtered, sorted device list endpoint. */
export async function listDevices(query: DeviceListQuery = {}): Promise<DeviceListResult> {
  await simulateLatency()

  const {
    search = '',
    status = 'all',
    type = 'all',
    technology = 'any',
    location = 'all',
    sortKey = 'device',
    sortDirection = 'asc',
    page = 1,
    pageSize = 15,
  } = query

  let results = DEVICES.filter((device) => matchesSearch(device, search))

  if (status !== 'all') {
    results = results.filter((device) => device.status === status)
  }
  if (type !== 'all') {
    results = results.filter((device) => device.type === type)
  }
  if (technology !== 'any') {
    results = results.filter((device) => device.technology === technology)
  }
  if (location !== 'all') {
    results = results.filter((device) => device.location === location)
  }

  results = results.slice().sort(compareDevices(sortKey, sortDirection))

  const total = results.length
  const start = (page - 1) * pageSize
  const items = results.slice(start, start + pageSize)

  return { items, total }
}

/** Simulates a single-resource fetch endpoint. Returns null rather than throwing when not found. */
export async function getDeviceById(id: string): Promise<Device | null> {
  await simulateLatency()
  return DEVICES.find((device) => device.id === id) ?? null
}

/**
 * Every device delivering a given service -- the Service Detail
 * Workspace's Devices section resolves this on demand rather than
 * Service carrying device detail itself (services/services/
 * serviceDataset.ts never stores it), the same on-demand-resolution
 * pattern Device Detail already uses for its own Assignment section.
 */
export async function listDevicesByServiceId(serviceId: string): Promise<Device[]> {
  await simulateLatency()
  return DEVICES.filter((device) => device.serviceId === serviceId)
}

/**
 * Distinct locations present in the dataset, for the Location filter.
 * Synchronous: small, static reference data, same reasoning as
 * customerRepository.ts's listAvailableCities.
 */
export function listAvailableLocations(): string[] {
  return Array.from(new Set(DEVICES.map((device) => device.location))).sort((a, b) => a.localeCompare(b))
}
