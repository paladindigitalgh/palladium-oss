import { ref, watch } from 'vue'
import type { TimelineEvent } from '@/types/timelineEvent'
import { listAllEvents } from '@/services/events/eventRepository'

export type ActivitySortDirection = 'asc' | 'desc'

const PAGE_SIZE = 25

function matchesSearch(event: TimelineEvent, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return (
    event.message.toLowerCase().includes(needle) ||
    event.type.toLowerCase().includes(needle) ||
    event.entityType.toLowerCase().includes(needle)
  )
}

/**
 * Owns every piece of state the Explorer Activity page needs and the
 * query orchestration around it -- the same shape useOLTCollection.ts
 * uses: GET /events (unbounded, see eventRepository.ts's listAllEvents)
 * has no server-side filtering, so every Event is fetched once per
 * change and search/sort/pagination happen client-side. Sorted newest
 * first by default (createdAt is ISO 8601, so a plain string compare
 * sorts correctly) -- the natural default for an activity feed, unlike
 * useOLTCollection's name-ascending default.
 */
export function useExplorerActivity() {
  const search = ref('')
  const sortDirection = ref<ActivitySortDirection>('desc')
  const page = ref(1)

  const events = ref<TimelineEvent[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  let requestId = 0

  async function fetchEvents() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const all = await listAllEvents()
      if (thisRequest !== requestId) return

      let results = all.filter((event) => matchesSearch(event, search.value))
      const direction = sortDirection.value === 'desc' ? -1 : 1
      results = results.slice().sort((a, b) => a.createdAt.localeCompare(b.createdAt) * direction)

      total.value = results.length
      const start = (page.value - 1) * PAGE_SIZE
      events.value = results.slice(start, start + PAGE_SIZE)
    } catch {
      if (thisRequest !== requestId) return
      error.value = true
    } finally {
      if (thisRequest === requestId) loading.value = false
    }
  }

  function toggleSort() {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  }

  watch(
    [search, sortDirection],
    () => {
      page.value = 1
      fetchEvents()
    },
    { immediate: true },
  )

  watch(page, fetchEvents)

  return {
    search,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    events,
    total,
    loading,
    error,
    retry: fetchEvents,
  }
}
