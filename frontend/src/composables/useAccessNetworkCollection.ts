import { ref, watch } from 'vue'
import type { AccessNetwork } from '@/types/accessNetwork'
import { listAccessNetworks, type AccessNetworkListQuery } from '@/services/accessNetworks/accessNetworkRepository'

export type AccessNetworkSortKey = NonNullable<AccessNetworkListQuery['sortKey']>
export type AccessNetworkSortDirection = NonNullable<AccessNetworkListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns every piece of state the Network Collection Workspace needs and
 * the query orchestration around it -- mirrors useCustomerCollection.ts
 * exactly (same reset-on-filter-change behavior).
 */
export function useAccessNetworkCollection() {
  const search = ref('')
  const status = ref<AccessNetwork['status'] | 'all'>('all')
  const sortKey = ref<AccessNetworkSortKey>('name')
  const sortDirection = ref<AccessNetworkSortDirection>('asc')
  const page = ref(1)

  const accessNetworks = ref<AccessNetwork[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  let requestId = 0

  async function fetchAccessNetworks() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const result = await listAccessNetworks({
        search: search.value,
        status: status.value,
        sortKey: sortKey.value,
        sortDirection: sortDirection.value,
        page: page.value,
        pageSize: PAGE_SIZE,
      })

      if (thisRequest !== requestId) return
      accessNetworks.value = result.items
      total.value = result.total
    } catch {
      if (thisRequest !== requestId) return
      error.value = true
    } finally {
      if (thisRequest === requestId) loading.value = false
    }
  }

  function toggleSort(key: AccessNetworkSortKey) {
    if (sortKey.value === key) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortKey.value = key
      sortDirection.value = 'asc'
    }
  }

  watch(
    [search, status, sortKey, sortDirection],
    () => {
      page.value = 1
      fetchAccessNetworks()
    },
    { immediate: true },
  )

  watch(page, fetchAccessNetworks)

  return {
    search,
    status,
    sortKey,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    accessNetworks,
    total,
    loading,
    error,
    retry: fetchAccessNetworks,
  }
}
