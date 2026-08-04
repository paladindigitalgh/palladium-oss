import { ref, watch } from 'vue'
import type { CustomerStatus, CustomerType, ServiceTechnology, Customer } from '@/types/customer'
import { listCustomers, listAvailableCities, type CustomerListQuery } from '@/services/customers/customerRepository'

export type CustomerSortKey = NonNullable<CustomerListQuery['sortKey']>
export type CustomerSortDirection = NonNullable<CustomerListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns every piece of state the Customer Collection Workspace needs and
 * the query orchestration around it (docs/11-COMPONENT-ARCHITECTURE.md,
 * "Separate business logic from presentation": this belongs in a
 * composable, not scattered through CustomerCollectionView.vue).
 *
 * Changing search, a filter, or sort resets to page 1 and re-queries;
 * changing the page alone re-queries without resetting. Because
 * customerRepository simulates network latency, a fast typist can have
 * multiple requests in flight at once -- `requestId` discards any
 * response that is no longer the most recent request, so a slow, stale
 * response can never overwrite a newer one.
 */
export function useCustomerCollection() {
  const search = ref('')
  const status = ref<CustomerStatus | 'all'>('active')
  const serviceTechnology = ref<ServiceTechnology | 'any'>('any')
  const customerType = ref<CustomerType | 'all'>('all')
  const city = ref<string>('all')
  const sortKey = ref<CustomerSortKey>('customer')
  const sortDirection = ref<CustomerSortDirection>('asc')
  const page = ref(1)

  const customers = ref<Customer[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  const cities = listAvailableCities()

  let requestId = 0

  async function fetchCustomers() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const result = await listCustomers({
        search: search.value,
        status: status.value,
        serviceTechnology: serviceTechnology.value,
        customerType: customerType.value,
        city: city.value,
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
    [search, status, serviceTechnology, customerType, city, sortKey, sortDirection],
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
    serviceTechnology,
    customerType,
    city,
    cities,
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
