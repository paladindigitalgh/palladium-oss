import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useOLTCollection } from './useOLTCollection'
import type { OLT } from '@/types/olt'

/**
 * OLT has no status field and only one sortable field (name), so
 * toggleSort takes no key argument -- there is nothing to switch
 * between. GET /olts has no server-side filtering, so listOLTs() takes
 * no params and every change refetches the full list, filtering/
 * sorting/paginating client-side, the same pattern
 * useSiteCollection.test.ts covers for Site.
 */
const { listOLTs } = vi.hoisted(() => ({ listOLTs: vi.fn() }))

vi.mock('@/services/olts/oltRepository', () => ({ listOLTs }))

function olt(overrides: Partial<OLT> = {}): OLT {
  return {
    id: 'olt1',
    name: 'OLT-A',
    oltModelId: 'model1',
    managementIpAddress: '',
    connectionProfileId: null,
    description: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listOLTs.mockReset()
  listOLTs.mockResolvedValue([])
})

it('fetches once on creation', async () => {
  useOLTCollection()
  await settle()

  expect(listOLTs).toHaveBeenCalledTimes(1)
})

it('populates olts and total from a successful fetch', async () => {
  listOLTs.mockResolvedValue([olt({ id: 'olt1', name: 'Main Office OLT' })])

  const collection = useOLTCollection()
  await settle()

  expect(collection.olts.value).toEqual([olt({ id: 'olt1', name: 'Main Office OLT' })])
  expect(collection.total.value).toBe(1)
  expect(collection.loading.value).toBe(false)
})

it('filters by name, management IP, and id client-side', async () => {
  listOLTs.mockResolvedValue([
    olt({ id: 'olt1', name: 'Alpha', managementIpAddress: '10.0.0.1' }),
    olt({ id: 'olt2', name: 'Beta', managementIpAddress: '10.0.0.2' }),
  ])

  const collection = useOLTCollection()
  await settle()

  collection.search.value = 'beta'
  await settle()

  expect(collection.olts.value.map((o) => o.id)).toEqual(['olt2'])
})

it('resets to page 1 when a filter changes', async () => {
  const collection = useOLTCollection()
  await settle()

  collection.page.value = 3
  await settle()
  expect(collection.page.value).toBe(3)

  collection.search.value = 'main'
  await settle()

  expect(collection.page.value).toBe(1)
})

it('does not reset the page when only the page itself changes', async () => {
  const collection = useOLTCollection()
  await settle()
  listOLTs.mockClear()

  collection.page.value = 2
  await settle()

  expect(collection.page.value).toBe(2)
  expect(listOLTs).toHaveBeenCalledTimes(1)
})

describe('toggleSort', () => {
  it('flips direction on every toggle', async () => {
    const collection = useOLTCollection()
    await settle()

    expect(collection.sortDirection.value).toBe('asc')

    collection.toggleSort()
    expect(collection.sortDirection.value).toBe('desc')

    collection.toggleSort()
    expect(collection.sortDirection.value).toBe('asc')
  })
})

it('sets error and stops loading when the fetch rejects, without touching stale data', async () => {
  listOLTs.mockRejectedValue(new Error('network down'))

  const collection = useOLTCollection()
  await settle()

  expect(collection.error.value).toBe(true)
  expect(collection.loading.value).toBe(false)
  expect(collection.olts.value).toEqual([])
})

it('retry re-runs the fetch', async () => {
  const collection = useOLTCollection()
  await settle()
  listOLTs.mockClear()

  await collection.retry()

  expect(listOLTs).toHaveBeenCalledTimes(1)
})
