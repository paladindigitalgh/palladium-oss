import type { Service, ServiceCategory } from '@/types/service'
import type { CustomerType, ServiceStatus, ServiceTechnology } from '@/types/customer'
import { SERVICES } from './serviceDataset'
import { DEVICES } from '@/services/devices/deviceDataset'

/**
 * The mock Service data source, mirroring services/customers/
 * customerRepository.ts and services/devices/deviceRepository.ts: the
 * Service Collection Workspace only ever calls listServices/
 * getServiceById, exactly what a real HTTP-backed implementation would
 * expose. Components never import SERVICES or serviceDataset.ts
 * directly.
 */

export interface ServiceListQuery {
  search?: string
  status?: ServiceStatus | 'all'
  technology?: ServiceTechnology | 'any'
  category?: ServiceCategory | 'all'
  customerType?: CustomerType | 'all'
  sortKey?: 'service' | 'customer' | 'technology' | 'status'
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface ServiceListResult {
  items: Service[]
  total: number
}

const SIMULATED_LATENCY_MS = 250

function simulateLatency(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, SIMULATED_LATENCY_MS))
}

function matchesSearch(service: Service, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  if (
    service.id.toLowerCase().includes(needle) ||
    service.customerName.toLowerCase().includes(needle) ||
    service.serviceAddress.toLowerCase().includes(needle) ||
    service.provisioningProfile.toLowerCase().includes(needle)
  ) {
    return true
  }
  // "Device serial" is a live join, not a field stored on Service --
  // Service never duplicates its devices' own data.
  return DEVICES.some((device) => device.serviceId === service.id && device.serialNumber.toLowerCase().includes(needle))
}

// A technician sorting by Status wants problems grouped together, not an
// alphabetical listing -- worst/most-actionable first under ascending,
// matching deviceRepository.ts's own STATUS_RANK convention.
const STATUS_RANK: Record<ServiceStatus, number> = {
  suspended: 0,
  provisioning: 1,
  cancelled: 2,
  active: 3,
}

function compareServices(
  sortKey: NonNullable<ServiceListQuery['sortKey']>,
  sortDirection: NonNullable<ServiceListQuery['sortDirection']>,
) {
  const direction = sortDirection === 'desc' ? -1 : 1
  return (a: Service, b: Service): number => {
    let comparison = 0
    switch (sortKey) {
      case 'customer':
        comparison = a.customerName.localeCompare(b.customerName)
        break
      case 'technology':
        comparison = a.technology.localeCompare(b.technology)
        break
      case 'status':
        comparison = STATUS_RANK[a.status] - STATUS_RANK[b.status]
        break
      case 'service':
      default:
        comparison = a.tier.localeCompare(b.tier)
    }
    return comparison * direction
  }
}

/** Simulates a paginated, filtered, sorted service list endpoint. */
export async function listServices(query: ServiceListQuery = {}): Promise<ServiceListResult> {
  await simulateLatency()

  const {
    search = '',
    status = 'all',
    technology = 'any',
    category = 'all',
    customerType = 'all',
    sortKey = 'service',
    sortDirection = 'asc',
    page = 1,
    pageSize = 15,
  } = query

  let results = SERVICES.filter((service) => matchesSearch(service, search))

  if (status !== 'all') {
    results = results.filter((service) => service.status === status)
  }
  if (technology !== 'any') {
    results = results.filter((service) => service.technology === technology)
  }
  if (category !== 'all') {
    results = results.filter((service) => service.category === category)
  }
  if (customerType !== 'all') {
    results = results.filter((service) => service.customerType === customerType)
  }

  results = results.slice().sort(compareServices(sortKey, sortDirection))

  const total = results.length
  const start = (page - 1) * pageSize
  const items = results.slice(start, start + pageSize)

  return { items, total }
}

/** Simulates a single-resource fetch endpoint. Returns null rather than throwing when not found. */
export async function getServiceById(id: string): Promise<Service | null> {
  await simulateLatency()
  return SERVICES.find((service) => service.id === id) ?? null
}
