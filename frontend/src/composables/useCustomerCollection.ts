import { ref, watch } from 'vue'
import type { Customer } from '@/types/customer'
import { listCustomers, type CustomerListQuery } from '@/services/customers/customerRepository'

export type CustomerSortKey = NonNullable<CustomerListQuery['sortKey']>
export type CustomerSortDirection = NonNullable<CustomerListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns every piece of state the Customer Collection Workspace needs and
 * the query orchestration around it. Trimmed to the filters the real
 * Customer domain actually supports (status, customer type) -- the
 * mock-era service-technology and city filters had no backend
 * equivalent and are gone, not faked.
 */
export function useCustomerCollection() {
  const search = ref('')
  const status = ref<Customer['status'] | 'all'>('all')
  const customerType = ref<Customer['customerType'] | 'all'>('all')
  const sortKey = ref<CustomerSortKey>('name')
  const sortDirection = ref<CustomerSortDirection>('asc')
  const page = ref(1)

  const customers = ref<Customer[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  let requestId = 0

  async function fetchCustomers() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const result = await listCustomers({
        search: search.value,
        status: status.value,
        customerType: customerType.value,
        sortKey: sortKey.value,
        sortDirection: sortDirection.value,
        page: page.value,
        pageSize: PAGE_SIZE,
      })

      if (thisRequest !== requestId) return
      customers.value = result.items
      total.value = result.total
    } catch {
      if (thisRequest !== requestId) return
      error.value = true
    } finally {
      if (thisRequest === requestId) loading.value = false
    }
  }

  function toggleSort(key: CustomerSortKey) {
    if (sortKey.value === key) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortKey.value = key
      sortDirection.value = 'asc'
    }
  }

  watch(
    [search, status, customerType, sortKey, sortDirection],
    () => {
      page.value = 1
      fetchCustomers()
    },
    { immediate: true },
  )

  watch(page, fetchCustomers)

  return {
    search,
    status,
    customerType,
    sortKey,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    customers,
    total,
    loading,
    error,
    retry: fetchCustomers,
  }
}
