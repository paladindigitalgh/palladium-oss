import { ref, watch } from 'vue'
import type { OLT } from '@/types/olt'
import { listOLTs } from '@/services/olts/oltRepository'

export type OLTSortDirection = 'asc' | 'desc'

const PAGE_SIZE = 15

function matchesSearch(olt: OLT, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return (
    olt.name.toLowerCase().includes(needle) ||
    olt.managementIpAddress.toLowerCase().includes(needle) ||
    olt.id.toLowerCase().includes(needle)
  )
}

/**
 * Owns every piece of state the Network Collection Workspace needs and
 * the query orchestration around it. GET /olts has no server-side
 * filtering (see oltRepository.ts), so every OLT is fetched once per
 * change and search/sort/pagination happen client-side, the same
 * pattern useSiteCollection.ts uses. OLT has no status filter and only
 * one sortable field, name, so there is no sortKey -- nothing to toggle
 * between.
 */
export function useOLTCollection() {
  const search = ref('')
  const sortDirection = ref<OLTSortDirection>('asc')
  const page = ref(1)

  const olts = ref<OLT[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  let requestId = 0

  async function fetchOLTs() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const all = await listOLTs()
      if (thisRequest !== requestId) return

      let results = all.filter((olt) => matchesSearch(olt, search.value))
      const direction = sortDirection.value === 'desc' ? -1 : 1
      results = results.slice().sort((a, b) => a.name.localeCompare(b.name) * direction)

      total.value = results.length
      const start = (page.value - 1) * PAGE_SIZE
      olts.value = results.slice(start, start + PAGE_SIZE)
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
      fetchOLTs()
    },
    { immediate: true },
  )

  watch(page, fetchOLTs)

  return {
    search,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    olts,
    total,
    loading,
    error,
    retry: fetchOLTs,
  }
}
