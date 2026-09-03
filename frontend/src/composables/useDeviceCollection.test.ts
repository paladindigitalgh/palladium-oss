import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useDeviceCollection } from './useDeviceCollection'

/** Mirrors useCustomerCollection.test.ts's shape exactly -- see that file. */
const { listDevices } = vi.hoisted(() => ({ listDevices: vi.fn() }))

vi.mock('@/services/devices/deviceRepository', () => ({ listDevices }))

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listDevices.mockReset()
  listDevices.mockResolvedValue({ items: [], total: 0 })
})

it('fetches once on creation with the default filters', async () => {
  useDeviceCollection()
  await settle()

  expect(listDevices).toHaveBeenCalledTimes(1)
  expect(listDevices).toHaveBeenCalledWith(
    expect.objectContaining({ search: '', status: 'all', sortKey: 'name', sortDirection: 'asc', page: 1 }),
  )
})

it('populates devices and total from a successful fetch', async () => {
  listDevices.mockResolvedValue({ items: [{ id: 'd1', name: 'ONT-1' }], total: 1 })

  const collection = useDeviceCollection()
  await settle()

  expect(collection.devices.value).toEqual([{ id: 'd1', name: 'ONT-1' }])
  expect(collection.total.value).toBe(1)
  expect(collection.loading.value).toBe(false)
})

it('resets to page 1 and refetches when a filter changes', async () => {
  const collection = useDeviceCollection()
  await settle()

  collection.page.value = 3
  await settle()
  expect(collection.page.value).toBe(3)

  collection.search.value = 'ont'
  await settle()

  expect(collection.page.value).toBe(1)
  expect(listDevices).toHaveBeenLastCalledWith(expect.objectContaining({ search: 'ont', page: 1 }))
})

it('does not reset the page when only the page itself changes', async () => {
  const collection = useDeviceCollection()
  await settle()
  listDevices.mockClear()

  collection.page.value = 2
  await settle()

  expect(collection.page.value).toBe(2)
  expect(listDevices).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }))
})

describe('toggleSort', () => {
  it('flips direction when toggling the same key', async () => {
    const collection = useDeviceCollection()
    await settle()

    expect(collection.sortKey.value).toBe('name')
    expect(collection.sortDirection.value).toBe('asc')

    collection.toggleSort('name')
    expect(collection.sortDirection.value).toBe('desc')

    collection.toggleSort('name')
    expect(collection.sortDirection.value).toBe('asc')
  })

  it('switches key and resets to ascending on a different key', async () => {
    const collection = useDeviceCollection()
    await settle()

    collection.toggleSort('status')

    expect(collection.sortKey.value).toBe('status')
    expect(collection.sortDirection.value).toBe('asc')
  })
})

it('sets error and stops loading when the fetch rejects, without touching stale data', async () => {
  listDevices.mockRejectedValue(new Error('network down'))

  const collection = useDeviceCollection()
  await settle()

  expect(collection.error.value).toBe(true)
  expect(collection.loading.value).toBe(false)
  expect(collection.devices.value).toEqual([])
})

it('retry re-runs the fetch with the current filters', async () => {
  const collection = useDeviceCollection()
  await settle()
  listDevices.mockClear()

  await collection.retry()

  expect(listDevices).toHaveBeenCalledTimes(1)
})
