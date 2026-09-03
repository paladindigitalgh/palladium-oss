import { ref, watch } from 'vue'
import type { Site } from '@/types/site'
import { listSites, type SiteListQuery } from '@/services/sites/siteRepository'

export type SiteSortDirection = NonNullable<SiteListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns every piece of state the Inventory Collection Workspace needs and
 * the query orchestration around it -- mirrors useAccessNetworkCollection.ts,
 * trimmed to what Site actually supports: no status filter, no sortKey
 * (Site has only one sortable field, name, so there is nothing to
 * toggle between).
 */
export function useSiteCollection() {
  const search = ref('')
  const sortDirection = ref<SiteSortDirection>('asc')
  const page = ref(1)

  const sites = ref<Site[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  let requestId = 0

  async function fetchSites() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const result = await listSites({
        search: search.value,
        sortDirection: sortDirection.value,
        page: page.value,
        pageSize: PAGE_SIZE,
      })

      if (thisRequest !== requestId) return
      sites.value = result.items
      total.value = result.total
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
      fetchSites()
    },
    { immediate: true },
  )

  watch(page, fetchSites)

  return {
    search,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    sites,
    total,
    loading,
    error,
    retry: fetchSites,
  }
}
