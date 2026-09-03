import { ref, watch } from 'vue'
import type { DeviceStatus, DeviceType } from '@/types/device'
import type { ServiceTechnology } from '@/types/mockCustomer'
import { listDevices, listAvailableLocations, type DeviceListQuery, type DeviceListResult } from '@/services/devices/deviceRepository'
import type { Device } from '@/types/device'

export type DeviceSortKey = NonNullable<DeviceListQuery['sortKey']>
export type DeviceSortDirection = NonNullable<DeviceListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns state and query orchestration for the Device Collection Workspace
 * -- mirrors composables/useCustomerCollection.ts exactly (same
 * reset-on-filter-change behavior, same requestId race guard against
 * customerRepository/deviceRepository's simulated latency). Kept as a
 * separate composable rather than a shared generic one: the two have
 * different filter shapes and no behavior would be saved by forcing them
 * through one abstraction (docs/11-COMPONENT-ARCHITECTURE.md, "Reuse
 * before creating" cuts both ways -- reuse what's genuinely the same,
 * don't force what merely looks similar).
 */
export function useDeviceCollection() {
  const search = ref('')
  const status = ref<DeviceStatus | 'all'>('all')
  const type = ref<DeviceType | 'all'>('all')
  const technology = ref<ServiceTechnology | 'any'>('any')
  const location = ref<string>('all')
  const sortKey = ref<DeviceSortKey>('device')
  const sortDirection = ref<DeviceSortDirection>('asc')
  const page = ref(1)

  const devices = ref<Device[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  const locations = listAvailableLocations()

  let requestId = 0

  async function fetchDevices() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const result: DeviceListResult = await listDevices({
        search: search.value,
        status: status.value,
        type: type.value,
        technology: technology.value,
        location: location.value,
        sortKey: sortKey.value,
        sortDirection: sortDirection.value,
        page: page.value,
        pageSize: PAGE_SIZE,
      })

      if (thisRequest !== requestId) return
      devices.value = result.items
      total.value = result.total
    } catch {
      if (thisRequest !== requestId) return
      error.value = true
    } finally {
      if (thisRequest === requestId) loading.value = false
    }
  }

  function toggleSort(key: DeviceSortKey) {
    if (sortKey.value === key) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortKey.value = key
      sortDirection.value = 'asc'
    }
  }

  watch(
    [search, status, type, technology, location, sortKey, sortDirection],
    () => {
      page.value = 1
      fetchDevices()
    },
    { immediate: true },
  )

  watch(page, fetchDevices)

  return {
    search,
    status,
    type,
    technology,
    location,
    locations,
    sortKey,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    devices,
    total,
    loading,
    error,
    retry: fetchDevices,
  }
}
