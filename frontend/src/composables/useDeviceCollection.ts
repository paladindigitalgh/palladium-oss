import { ref, watch } from 'vue'
import type { Device } from '@/types/device'
import { listDevices, type DeviceListQuery } from '@/services/devices/deviceRepository'

export type DeviceSortKey = NonNullable<DeviceListQuery['sortKey']>
export type DeviceSortDirection = NonNullable<DeviceListQuery['sortDirection']>

const PAGE_SIZE = 15

/**
 * Owns state and query orchestration for the Device Collection Workspace
 * -- mirrors composables/useCustomerCollection.ts exactly. Trimmed to the
 * filters the real Device domain actually supports (status) -- the
 * mock-era type/technology/location filters had no backend equivalent
 * and are gone, not faked.
 */
export function useDeviceCollection() {
  const search = ref('')
  const status = ref<Device['status'] | 'all'>('all')
  const sortKey = ref<DeviceSortKey>('name')
  const sortDirection = ref<DeviceSortDirection>('asc')
  const page = ref(1)

  const devices = ref<Device[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(false)

  let requestId = 0

  async function fetchDevices() {
    const thisRequest = ++requestId
    loading.value = true
    error.value = false

    try {
      const result = await listDevices({
        search: search.value,
        status: status.value,
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
    [search, status, sortKey, sortDirection],
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
