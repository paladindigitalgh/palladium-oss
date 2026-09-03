import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useAccessNetworkCollection } from './useAccessNetworkCollection'

/** Mirrors useCustomerCollection.test.ts exactly -- same state-orchestration cases, adjusted field names. */
const { listAccessNetworks } = vi.hoisted(() => ({ listAccessNetworks: vi.fn() }))

vi.mock('@/services/accessNetworks/accessNetworkRepository', () => ({ listAccessNetworks }))

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listAccessNetworks.mockReset()
  listAccessNetworks.mockResolvedValue({ items: [], total: 0 })
})

it('fetches once on creation with the default filters', async () => {
  useAccessNetworkCollection()
  await settle()

  expect(listAccessNetworks).toHaveBeenCalledTimes(1)
  expect(listAccessNetworks).toHaveBeenCalledWith(
    expect.objectContaining({ search: '', status: 'all', sortKey: 'name', sortDirection: 'asc', page: 1 }),
  )
})

it('populates accessNetworks and total from a successful fetch', async () => {
  listAccessNetworks.mockResolvedValue({ items: [{ id: 'an1', name: 'Metro North' }], total: 1 })

  const collection = useAccessNetworkCollection()
  await settle()

  expect(collection.accessNetworks.value).toEqual([{ id: 'an1', name: 'Metro North' }])
  expect(collection.total.value).toBe(1)
  expect(collection.loading.value).toBe(false)
})

it('resets to page 1 and refetches when a filter changes', async () => {
  const collection = useAccessNetworkCollection()
  await settle()

  collection.page.value = 3
  await settle()
  expect(collection.page.value).toBe(3)

  collection.search.value = 'metro'
  await settle()

  expect(collection.page.value).toBe(1)
  expect(listAccessNetworks).toHaveBeenLastCalledWith(expect.objectContaining({ search: 'metro', page: 1 }))
})

it('does not reset the page when only the page itself changes', async () => {
  const collection = useAccessNetworkCollection()
  await settle()
  listAccessNetworks.mockClear()

  collection.page.value = 2
  await settle()

  expect(collection.page.value).toBe(2)
  expect(listAccessNetworks).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }))
})

describe('toggleSort', () => {
  it('flips direction when toggling the same key', async () => {
    const collection = useAccessNetworkCollection()
    await settle()

    expect(collection.sortKey.value).toBe('name')
    expect(collection.sortDirection.value).toBe('asc')

    collection.toggleSort('name')
    expect(collection.sortDirection.value).toBe('desc')

    collection.toggleSort('name')
    expect(collection.sortDirection.value).toBe('asc')
  })

  it('switches key and resets to ascending on a different key', async () => {
    const collection = useAccessNetworkCollection()
    await settle()

    collection.toggleSort('status')

    expect(collection.sortKey.value).toBe('status')
    expect(collection.sortDirection.value).toBe('asc')
  })
})

it('sets error and stops loading when the fetch rejects, without touching stale data', async () => {
  listAccessNetworks.mockRejectedValue(new Error('network down'))

  const collection = useAccessNetworkCollection()
  await settle()

  expect(collection.error.value).toBe(true)
  expect(collection.loading.value).toBe(false)
  expect(collection.accessNetworks.value).toEqual([])
})

it('retry re-runs the fetch with the current filters', async () => {
  const collection = useAccessNetworkCollection()
  await settle()
  listAccessNetworks.mockClear()

  await collection.retry()

  expect(listAccessNetworks).toHaveBeenCalledTimes(1)
})
