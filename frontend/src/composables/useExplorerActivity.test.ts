import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useExplorerActivity } from './useExplorerActivity'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * Mirrors useOLTCollection.test.ts's shape: GET /events (unbounded) has
 * no server-side filtering, so listAllEvents() takes no params and every
 * change refetches the full list, filtering/sorting/paginating
 * client-side. Unlike OLT, the default sort is newest-first (createdAt
 * desc), the natural default for an activity feed.
 */
const { listAllEvents } = vi.hoisted(() => ({ listAllEvents: vi.fn() }))

vi.mock('@/services/events/eventRepository', () => ({ listAllEvents }))

function event(overrides: Partial<TimelineEvent> = {}): TimelineEvent {
  return {
    id: 'e1',
    entityType: 'device',
    entityId: 'd1',
    type: 'device.created',
    message: 'Created device test-15',
    metadata: null,
    actorUserId: null,
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listAllEvents.mockReset()
  listAllEvents.mockResolvedValue([])
})

it('fetches once on creation', async () => {
  useExplorerActivity()
  await settle()

  expect(listAllEvents).toHaveBeenCalledTimes(1)
})

it('populates events and total from a successful fetch', async () => {
  listAllEvents.mockResolvedValue([event({ id: 'e1', message: 'Created customer Acme Corp' })])

  const activity = useExplorerActivity()
  await settle()

  expect(activity.events.value).toEqual([event({ id: 'e1', message: 'Created customer Acme Corp' })])
  expect(activity.total.value).toBe(1)
  expect(activity.loading.value).toBe(false)
})

it('sorts newest first by default', async () => {
  listAllEvents.mockResolvedValue([
    event({ id: 'old', createdAt: '2026-01-01T00:00:00Z' }),
    event({ id: 'new', createdAt: '2026-01-02T00:00:00Z' }),
  ])

  const activity = useExplorerActivity()
  await settle()

  expect(activity.events.value.map((e) => e.id)).toEqual(['new', 'old'])
})

it('filters by message, type, and entityType client-side', async () => {
  listAllEvents.mockResolvedValue([
    event({ id: 'e1', message: 'Created device test-15', type: 'device.created', entityType: 'device' }),
    event({ id: 'e2', message: 'Created customer Acme Corp', type: 'customer.created', entityType: 'customer' }),
  ])

  const activity = useExplorerActivity()
  await settle()

  activity.search.value = 'acme'
  await settle()

  expect(activity.events.value.map((e) => e.id)).toEqual(['e2'])
})

it('resets to page 1 when a filter changes', async () => {
  const activity = useExplorerActivity()
  await settle()

  activity.page.value = 3
  await settle()
  expect(activity.page.value).toBe(3)

  activity.search.value = 'device'
  await settle()

  expect(activity.page.value).toBe(1)
})

it('does not reset the page when only the page itself changes', async () => {
  const activity = useExplorerActivity()
  await settle()
  listAllEvents.mockClear()

  activity.page.value = 2
  await settle()

  expect(activity.page.value).toBe(2)
  expect(listAllEvents).toHaveBeenCalledTimes(1)
})

describe('toggleSort', () => {
  it('flips direction on every toggle, starting from desc', async () => {
    const activity = useExplorerActivity()
    await settle()

    expect(activity.sortDirection.value).toBe('desc')

    activity.toggleSort()
    expect(activity.sortDirection.value).toBe('asc')

    activity.toggleSort()
    expect(activity.sortDirection.value).toBe('desc')
  })
})

it('sets error and stops loading when the fetch rejects, without touching stale data', async () => {
  listAllEvents.mockRejectedValue(new Error('network down'))

  const activity = useExplorerActivity()
  await settle()

  expect(activity.error.value).toBe(true)
  expect(activity.loading.value).toBe(false)
  expect(activity.events.value).toEqual([])
})

it('retry re-runs the fetch', async () => {
  const activity = useExplorerActivity()
  await settle()
  listAllEvents.mockClear()

  await activity.retry()

  expect(listAllEvents).toHaveBeenCalledTimes(1)
})
