import type { Customer, CustomerStatus, CustomerType, ServiceTechnology } from '@/types/customer'
import { CUSTOMERS } from './customerDataset'

/**
 * The mock Customer data source. This is written and consumed as a
 * legitimate service-layer boundary, not a placeholder array a component
 * reaches into: CustomerCollectionView (and later the Customer Detail
 * Workspace) only ever call listCustomers/getCustomerById/
 * listAvailableCities, exactly the shape a real HTTP-backed
 * implementation would expose. Replacing the body of this file with
 * `fetch('/api/v1/customers?...')` calls should require no changes
 * outside this file.
 */

export interface CustomerListQuery {
  search?: string
  status?: CustomerStatus | 'all'
  serviceTechnology?: ServiceTechnology | 'any'
  customerType?: CustomerType | 'all'
  city?: string | 'all'
  sortKey?: 'customer' | 'location' | 'primaryService'
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface CustomerListResult {
  items: Customer[]
  total: number
}

// A real backend has network latency; simulating a small amount here
// keeps the Collection Workspace's loading state genuinely exercised
// during development instead of only in theory.
const SIMULATED_LATENCY_MS = 250

function simulateLatency(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, SIMULATED_LATENCY_MS))
}

function matchesSearch(customer: Customer, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return (
    customer.name.toLowerCase().includes(needle) ||
    customer.id.toLowerCase().includes(needle) ||
    customer.address.toLowerCase().includes(needle) ||
    customer.city.toLowerCase().includes(needle)
  )
}

function compareCustomers(
  sortKey: NonNullable<CustomerListQuery['sortKey']>,
  sortDirection: NonNullable<CustomerListQuery['sortDirection']>,
) {
  const direction = sortDirection === 'desc' ? -1 : 1
  return (a: Customer, b: Customer): number => {
    let comparison = 0
    switch (sortKey) {
      case 'location':
        comparison = a.city.localeCompare(b.city) || a.name.localeCompare(b.name)
        break
      case 'primaryService':
        comparison =
          a.services[0].technology.localeCompare(b.services[0].technology) ||
          a.services[0].tier.localeCompare(b.services[0].tier)
        break
      case 'customer':
      default:
        comparison = a.name.localeCompare(b.name)
    }
    return comparison * direction
  }
}

/** Simulates a paginated, filtered, sorted customer list endpoint. */
export async function listCustomers(query: CustomerListQuery = {}): Promise<CustomerListResult> {
  await simulateLatency()

  const {
    search = '',
    status = 'active',
    serviceTechnology = 'any',
    customerType = 'all',
    city = 'all',
    sortKey = 'customer',
    sortDirection = 'asc',
    page = 1,
    pageSize = 15,
  } = query

  let results = CUSTOMERS.filter((customer) => matchesSearch(customer, search))

  if (status !== 'all') {
    results = results.filter((customer) => customer.status === status)
  }
  if (serviceTechnology !== 'any') {
    results = results.filter((customer) => customer.services.some((service) => service.technology === serviceTechnology))
  }
  if (customerType !== 'all') {
    results = results.filter((customer) => customer.type === customerType)
  }
  if (city !== 'all') {
    results = results.filter((customer) => customer.city === city)
  }

  results = results.slice().sort(compareCustomers(sortKey, sortDirection))

  const total = results.length
  const start = (page - 1) * pageSize
  const items = results.slice(start, start + pageSize)

  return { items, total }
}

/** Simulates a single-resource fetch endpoint. Returns null rather than throwing when not found. */
export async function getCustomerById(id: string): Promise<Customer | null> {
  await simulateLatency()
  return CUSTOMERS.find((customer) => customer.id === id) ?? null
}

/**
 * Distinct cities present in the dataset, for the Location filter.
 * Synchronous: this is small, static reference data (closer to an enum
 * than a paginated resource), so it does not need the same async/loading
 * treatment as listCustomers.
 */
export function listAvailableCities(): string[] {
  return Array.from(new Set(CUSTOMERS.map((customer) => customer.city))).sort((a, b) => a.localeCompare(b))
}
