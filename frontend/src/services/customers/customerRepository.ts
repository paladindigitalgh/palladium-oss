import type { Customer } from '@/types/customer'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real Customer data source, replacing the mock dataset this file
 * used to read from. GET /customers has no server-side filtering (see
 * internal/customer/httpapi), so search/sort/pagination happen
 * client-side over the fetched list -- the same shape the Customer
 * Collection Workspace already expected, just backed by a real fetch.
 */

interface CustomerDto {
  id: string
  name: string
  customer_type: Customer['customerType']
  status: Customer['status']
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: CustomerDto): Customer {
  return {
    id: dto.id,
    name: dto.name,
    customerType: dto.customer_type,
    status: dto.status,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export interface CustomerListQuery {
  search?: string
  status?: Customer['status'] | 'all'
  customerType?: Customer['customerType'] | 'all'
  sortKey?: 'name' | 'status'
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface CustomerListResult {
  items: Customer[]
  total: number
}

function matchesSearch(customer: Customer, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return customer.name.toLowerCase().includes(needle) || customer.id.toLowerCase().includes(needle)
}

function compareCustomers(sortKey: NonNullable<CustomerListQuery['sortKey']>, direction: number) {
  return (a: Customer, b: Customer): number => {
    const comparison = sortKey === 'status' ? a.status.localeCompare(b.status) : a.name.localeCompare(b.name)
    return comparison * direction
  }
}

/** Fetches every Customer and applies search/filter/sort/pagination client-side. */
export async function listCustomers(query: CustomerListQuery = {}): Promise<CustomerListResult> {
  const {
    search = '',
    status = 'all',
    customerType = 'all',
    sortKey = 'name',
    sortDirection = 'asc',
    page = 1,
    pageSize = 15,
  } = query

  const { customers } = await apiFetch<{ customers: CustomerDto[] }>('/customers/')
  let results = customers.map(fromDto).filter((customer) => matchesSearch(customer, search))

  if (status !== 'all') results = results.filter((customer) => customer.status === status)
  if (customerType !== 'all') results = results.filter((customer) => customer.customerType === customerType)

  results = results.slice().sort(compareCustomers(sortKey, sortDirection === 'desc' ? -1 : 1))

  const total = results.length
  const start = (page - 1) * pageSize
  return { items: results.slice(start, start + pageSize), total }
}

/** Fetches a single Customer, returning null (not throwing) when it does not exist. */
export async function getCustomerById(id: string): Promise<Customer | null> {
  try {
    const dto = await apiFetch<CustomerDto>(`/customers/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateCustomerInput {
  name: string
  customerType: Customer['customerType']
  status: Customer['status']
  description: string
}

export async function createCustomer(input: CreateCustomerInput): Promise<Customer> {
  const dto = await apiFetch<CustomerDto>('/customers/', {
    method: 'POST',
    body: { name: input.name, customer_type: input.customerType, status: input.status, description: input.description },
  })
  return fromDto(dto)
}

export interface UpdateCustomerInput {
  name: string
  customerType: Customer['customerType']
  status: Customer['status']
  description: string
}

export async function updateCustomer(id: string, input: UpdateCustomerInput): Promise<Customer> {
  const dto = await apiFetch<CustomerDto>(`/customers/${id}`, {
    method: 'PUT',
    body: { name: input.name, customer_type: input.customerType, status: input.status, description: input.description },
  })
  return fromDto(dto)
}

/**
 * Deletes the Customer identified by id. customers.id is referenced by
 * locations.customer_id ON DELETE RESTRICT, so this throws an ApiError
 * with kind "conflict" if the customer still has any Location -- callers
 * should catch that and explain it, not treat it as an unexpected
 * failure.
 */
export async function deleteCustomer(id: string): Promise<void> {
  await apiFetch<void>(`/customers/${id}`, { method: 'DELETE' })
}
