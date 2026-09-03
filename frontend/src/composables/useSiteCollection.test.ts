import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useSiteCollection } from './useSiteCollection'

/**
 * Mirrors useAccessNetworkCollection.test.ts, trimmed to what Site
 * actually supports: no status, and toggleSort takes no key argument
 * since name is the only sortable field.
 */
const { listSites } = vi.hoisted(() => ({ listSites: vi.fn() }))

vi.mock('@/services/sites/siteRepository', () => ({ listSites }))

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listSites.mockReset()
  listSites.mockResolvedValue({ items: [], total: 0 })
})

it('fetches once on creation with the default filters', async () => {
  useSiteCollection()
  await settle()

  expect(listSites).toHaveBeenCalledTimes(1)
  expect(listSites).toHaveBeenCalledWith(expect.objectContaining({ search: '', sortDirection: 'asc', page: 1 }))
})

it('populates sites and total from a successful fetch', async () => {
  listSites.mockResolvedValue({ items: [{ id: 's1', name: 'Main Office' }], total: 1 })

  const collection = useSiteCollection()
  await settle()

  expect(collection.sites.value).toEqual([{ id: 's1', name: 'Main Office' }])
  expect(collection.total.value).toBe(1)
  expect(collection.loading.value).toBe(false)
})

it('resets to page 1 and refetches when a filter changes', async () => {
  const collection = useSiteCollection()
  await settle()

  collection.page.value = 3
  await settle()
  expect(collection.page.value).toBe(3)

  collection.search.value = 'main'
  await settle()

  expect(collection.page.value).toBe(1)
  expect(listSites).toHaveBeenLastCalledWith(expect.objectContaining({ search: 'main', page: 1 }))
})

it('does not reset the page when only the page itself changes', async () => {
  const collection = useSiteCollection()
  await settle()
  listSites.mockClear()

  collection.page.value = 2
  await settle()

  expect(collection.page.value).toBe(2)
  expect(listSites).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }))
})

describe('toggleSort', () => {
  it('flips direction on every toggle', async () => {
    const collection = useSiteCollection()
    await settle()

    expect(collection.sortDirection.value).toBe('asc')

    collection.toggleSort()
    expect(collection.sortDirection.value).toBe('desc')

    collection.toggleSort()
    expect(collection.sortDirection.value).toBe('asc')
  })
})

it('sets error and stops loading when the fetch rejects, without touching stale data', async () => {
  listSites.mockRejectedValue(new Error('network down'))

  const collection = useSiteCollection()
  await settle()

  expect(collection.error.value).toBe(true)
  expect(collection.loading.value).toBe(false)
  expect(collection.sites.value).toEqual([])
})

it('retry re-runs the fetch with the current filters', async () => {
  const collection = useSiteCollection()
  await settle()
  listSites.mockClear()

  await collection.retry()

  expect(listSites).toHaveBeenCalledTimes(1)
})
