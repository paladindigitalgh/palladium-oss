import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useCustomerCollection } from './useCustomerCollection'

/**
 * The reference test for every use*Collection.ts composable (customer/
 * device share this exact shape): mock the repository, not apiFetch --
 * the composable's own job is state orchestration (loading/error/page
 * reset on filter change), not fetching, so that is what this tests.
 * Two `await nextTick()`s after any state change: one lets the `watch`
 * callback run, the second lets the mocked repository's resolved Promise
 * settle and the `.then` continuation apply back to the refs.
 */
const { listCustomers } = vi.hoisted(() => ({ listCustomers: vi.fn() }))

vi.mock('@/services/customers/customerRepository', () => ({ listCustomers }))

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listCustomers.mockReset()
  listCustomers.mockResolvedValue({ items: [], total: 0 })
})

it('fetches once on creation with the default filters', async () => {
  useCustomerCollection()
  await settle()

  expect(listCustomers).toHaveBeenCalledTimes(1)
  expect(listCustomers).toHaveBeenCalledWith(
    expect.objectContaining({ search: '', status: 'all', customerType: 'all', sortKey: 'name', sortDirection: 'asc', page: 1 }),
  )
})

it('populates customers and total from a successful fetch', async () => {
  listCustomers.mockResolvedValue({ items: [{ id: 'c1', name: 'Acme' }], total: 1 })

  const collection = useCustomerCollection()
  await settle()

  expect(collection.customers.value).toEqual([{ id: 'c1', name: 'Acme' }])
  expect(collection.total.value).toBe(1)
  expect(collection.loading.value).toBe(false)
})

it('resets to page 1 and refetches when a filter changes', async () => {
  const collection = useCustomerCollection()
  await settle()

  collection.page.value = 3
  await settle()
  expect(collection.page.value).toBe(3)

  collection.search.value = 'acme'
  await settle()

  expect(collection.page.value).toBe(1)
  expect(listCustomers).toHaveBeenLastCalledWith(expect.objectContaining({ search: 'acme', page: 1 }))
})

it('does not reset the page when only the page itself changes', async () => {
  const collection = useCustomerCollection()
  await settle()
  listCustomers.mockClear()

  collection.page.value = 2
  await settle()

  expect(collection.page.value).toBe(2)
  expect(listCustomers).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }))
})

describe('toggleSort', () => {
  it('flips direction when toggling the same key', async () => {
    const collection = useCustomerCollection()
    await settle()

    expect(collection.sortKey.value).toBe('name')
    expect(collection.sortDirection.value).toBe('asc')

    collection.toggleSort('name')
    expect(collection.sortDirection.value).toBe('desc')

    collection.toggleSort('name')
    expect(collection.sortDirection.value).toBe('asc')
  })

  it('switches key and resets to ascending on a different key', async () => {
    const collection = useCustomerCollection()
    await settle()

    collection.toggleSort('status')

    expect(collection.sortKey.value).toBe('status')
    expect(collection.sortDirection.value).toBe('asc')
  })
})

it('sets error and stops loading when the fetch rejects, without touching stale data', async () => {
  listCustomers.mockRejectedValue(new Error('network down'))

  const collection = useCustomerCollection()
  await settle()

  expect(collection.error.value).toBe(true)
  expect(collection.loading.value).toBe(false)
  expect(collection.customers.value).toEqual([])
})

it('retry re-runs the fetch with the current filters', async () => {
  const collection = useCustomerCollection()
  await settle()
  listCustomers.mockClear()

  await collection.retry()

  expect(listCustomers).toHaveBeenCalledTimes(1)
})
