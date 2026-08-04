import { ref, watch } from 'vue'
import type { CustomerType, ServiceStatus, ServiceTechnology } from '@/types/customer'
import type { ServiceCategory } from '@/types/service'
import type { Service } from '@/types/service'
import { listServices, type ServiceListQuery, type ServiceListResult } from '@/services/services/serviceRepository'

export type ServiceSortKey = NonNullable<ServiceListQuery['sortKey']>
export type ServiceSortDirection = NonNullable<ServiceListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns state and query orchestration for the Service Collection
 * Workspace -- mirrors composables/useCustomerCollection.ts and
 * useDeviceCollection.ts exactly (same reset-on-filter-change behavior,
 * same requestId race guard against serviceRepository's simulated
 * latency).
 */
export function useServiceCollection() {
  const search = ref('')
  const status = ref<ServiceStatus | 'all'>('all')
  const technology = ref<ServiceTechnology | 'any'>('any')
  const category = ref<ServiceCategory | 'all'>('all')
  const customerType = ref<CustomerType | 'all'>('all')
  const sortKey = ref<ServiceSortKey>('service')
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
      const result: ServiceListResult = await listServices({
        search: search.value,
        status: status.value,
        technology: technology.value,
        category: category.value,
        customerType: customerType.value,
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
    [search, status, technology, category, customerType, sortKey, sortDirection],
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
    technology,
    category,
    customerType,
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
