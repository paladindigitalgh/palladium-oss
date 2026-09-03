import { ref, watch } from 'vue'
import type { Service } from '@/types/service'
import { listServices, type ServiceListQuery } from '@/services/services/serviceRepository'

export type ServiceSortKey = NonNullable<ServiceListQuery['sortKey']>
export type ServiceSortDirection = NonNullable<ServiceListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns state and query orchestration for the Service Collection
 * Workspace, mirroring useCustomerCollection.ts. Trimmed to the filters
 * the real Service domain actually supports (status only) -- the
 * mock-era technology/category/customer-type filters had no backend
 * equivalent and are gone, not faked.
 */
export function useServiceCollection() {
  const search = ref('')
  const status = ref<Service['status'] | 'all'>('all')
  const sortKey = ref<ServiceSortKey>('id')
  const sortDirection = ref<ServiceSortDirection>('asc')
  const page = ref(1)

  const services = ref<Service[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  let requestId = 0

  async function fetchServices() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const result = await listServices({
        search: search.value,
        status: status.value,
        sortKey: sortKey.value,
        sortDirection: sortDirection.value,
        page: page.value,
        pageSize: PAGE_SIZE,
      })

      if (thisRequest !== requestId) return
      services.value = result.items
      total.value = result.total
    } catch {
      if (thisRequest !== requestId) return
      error.value = true
    } finally {
      if (thisRequest === requestId) loading.value = false
    }
  }

  function toggleSort(key: ServiceSortKey) {
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
      fetchServices()
    },
    { immediate: true },
  )

  watch(page, fetchServices)

  return {
    search,
    status,
    sortKey,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    services,
    total,
    loading,
    error,
    retry: fetchServices,
  }
}
