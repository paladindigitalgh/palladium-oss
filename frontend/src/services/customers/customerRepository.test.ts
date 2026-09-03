import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ApiError } from '@/services/api/httpClient'
import { listCustomers, getCustomerById, createCustomer, updateCustomer, deleteCustomer } from './customerRepository'

/**
 * The reference test for every *Repository.ts module in this codebase
 * (customer/service/device all share this exact shape): mock apiFetch,
 * not fetch itself, since apiFetch is the one seam every repository is
 * built on (see services/api/httpClient.ts). Covers what actually varies
 * between repositories -- the client-side search/filter/sort/pagination
 * logic and the wire shape sent to the API -- not apiFetch itself, which
 * has no test of its own yet but is simple enough to trust for now.
 */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function customerDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'c1',
    name: 'Acme',
    customer_type: 'Business',
    status: 'Active',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listCustomers', () => {
  it('filters by search term across name and id', async () => {
    apiFetch.mockResolvedValue({ customers: [customerDto({ id: 'c1', name: 'Acme' }), customerDto({ id: 'c2', name: 'Globex' })] })

    const result = await listCustomers({ search: 'acm' })

    expect(result.items.map((c) => c.id)).toEqual(['c1'])
  })

  it('filters by status', async () => {
    apiFetch.mockResolvedValue({
      customers: [customerDto({ id: 'c1', status: 'Active' }), customerDto({ id: 'c2', status: 'Archived' })],
    })

    const result = await listCustomers({ status: 'Archived' })

    expect(result.items.map((c) => c.id)).toEqual(['c2'])
  })

  it('sorts by name ascending by default', async () => {
    apiFetch.mockResolvedValue({ customers: [customerDto({ id: 'c1', name: 'Zeta' }), customerDto({ id: 'c2', name: 'Alpha' })] })

    const result = await listCustomers()

    expect(result.items.map((c) => c.name)).toEqual(['Alpha', 'Zeta'])
  })

  it('reverses sort direction on request', async () => {
    apiFetch.mockResolvedValue({ customers: [customerDto({ id: 'c1', name: 'Alpha' }), customerDto({ id: 'c2', name: 'Zeta' })] })

    const result = await listCustomers({ sortDirection: 'desc' })

    expect(result.items.map((c) => c.name)).toEqual(['Zeta', 'Alpha'])
  })

  it('paginates results while reporting the true total', async () => {
    const customers = Array.from({ length: 20 }, (_, i) => customerDto({ id: `c${i}`, name: `Customer ${i}` }))
    apiFetch.mockResolvedValue({ customers })

    const result = await listCustomers({ page: 2, pageSize: 15 })

    expect(result.total).toBe(20)
    expect(result.items).toHaveLength(5)
  })
})

describe('getCustomerById', () => {
  it('returns the customer when found', async () => {
    apiFetch.mockResolvedValue(customerDto({ id: 'c1' }))

    const result = await getCustomerById('c1')

    expect(result?.id).toBe('c1')
  })

  it('returns null instead of throwing when the customer does not exist', async () => {
    apiFetch.mockRejectedValue(new ApiError('not found', 'not_found', 404))

    const result = await getCustomerById('missing')

    expect(result).toBeNull()
  })

  it('rethrows any error that is not a not_found', async () => {
    apiFetch.mockRejectedValue(new ApiError('boom', 'internal', 500))

    await expect(getCustomerById('c1')).rejects.toThrow('boom')
  })
})

describe('createCustomer', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(customerDto({ id: 'new' }))

    await createCustomer({ name: 'Acme', customerType: 'Business', status: 'Active', description: 'A widget maker' })

    expect(apiFetch).toHaveBeenCalledWith('/customers/', {
      method: 'POST',
      body: { name: 'Acme', customer_type: 'Business', status: 'Active', description: 'A widget maker' },
    })
  })
})

describe('updateCustomer', () => {
  it('sends a PUT with the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(customerDto({ id: 'c1', name: 'Acme Renamed' }))

    await updateCustomer('c1', { name: 'Acme Renamed', customerType: 'Business', status: 'Inactive', description: 'Updated' })

    expect(apiFetch).toHaveBeenCalledWith('/customers/c1', {
      method: 'PUT',
      body: { name: 'Acme Renamed', customer_type: 'Business', status: 'Inactive', description: 'Updated' },
    })
  })
})

describe('deleteCustomer', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteCustomer('c1')

    expect(apiFetch).toHaveBeenCalledWith('/customers/c1', { method: 'DELETE' })
  })
})
