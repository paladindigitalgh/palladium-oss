import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useServiceCollection } from './useServiceCollection'

/** Mirrors useCustomerCollection.test.ts's shape exactly -- see that file. */
const { listServices } = vi.hoisted(() => ({ listServices: vi.fn() }))

vi.mock('@/services/services/serviceRepository', () => ({ listServices }))

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listServices.mockReset()
  listServices.mockResolvedValue({ items: [], total: 0 })
})

it('fetches once on creation with the default filters', async () => {
  useServiceCollection()
  await settle()

  expect(listServices).toHaveBeenCalledTimes(1)
  expect(listServices).toHaveBeenCalledWith(
    expect.objectContaining({ search: '', status: 'all', sortKey: 'id', sortDirection: 'asc', page: 1 }),
  )
})

it('populates services and total from a successful fetch', async () => {
  listServices.mockResolvedValue({ items: [{ id: 's1' }], total: 1 })

  const collection = useServiceCollection()
  await settle()

  expect(collection.services.value).toEqual([{ id: 's1' }])
  expect(collection.total.value).toBe(1)
  expect(collection.loading.value).toBe(false)
})

it('resets to page 1 and refetches when a filter changes', async () => {
  const collection = useServiceCollection()
  await settle()

  collection.page.value = 3
  await settle()
  expect(collection.page.value).toBe(3)

  collection.search.value = 's1'
  await settle()

  expect(collection.page.value).toBe(1)
  expect(listServices).toHaveBeenLastCalledWith(expect.objectContaining({ search: 's1', page: 1 }))
})

it('does not reset the page when only the page itself changes', async () => {
  const collection = useServiceCollection()
  await settle()
  listServices.mockClear()

  collection.page.value = 2
  await settle()

  expect(collection.page.value).toBe(2)
  expect(listServices).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }))
})

describe('toggleSort', () => {
  it('flips direction when toggling the same key', async () => {
    const collection = useServiceCollection()
    await settle()

    expect(collection.sortKey.value).toBe('id')
    expect(collection.sortDirection.value).toBe('asc')

    collection.toggleSort('id')
    expect(collection.sortDirection.value).toBe('desc')

    collection.toggleSort('id')
    expect(collection.sortDirection.value).toBe('asc')
  })

  it('switches key and resets to ascending on a different key', async () => {
    const collection = useServiceCollection()
    await settle()

    collection.toggleSort('status')

    expect(collection.sortKey.value).toBe('status')
    expect(collection.sortDirection.value).toBe('asc')
  })
})

it('sets error and stops loading when the fetch rejects, without touching stale data', async () => {
  listServices.mockRejectedValue(new Error('network down'))

  const collection = useServiceCollection()
  await settle()

  expect(collection.error.value).toBe(true)
  expect(collection.loading.value).toBe(false)
  expect(collection.services.value).toEqual([])
})

it('retry re-runs the fetch with the current filters', async () => {
  const collection = useServiceCollection()
  await settle()
  listServices.mockClear()

  await collection.retry()

  expect(listServices).toHaveBeenCalledTimes(1)
})
