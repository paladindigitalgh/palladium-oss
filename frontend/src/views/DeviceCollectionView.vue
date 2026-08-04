<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import type { Device, DeviceStatus } from '@/types/device'
import { useDeviceCollection, type DeviceSortKey } from '@/composables/useDeviceCollection'

/**
 * The Device Collection View (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * "Collection View & Detail View"; section 10, "Device Workspace"):
 * discovery only, for finding managed equipment on the live network --
 * not an inventory of unassigned stock (that is a future, separate
 * workspace). Built entirely from the same Collection Workspace
 * components the Customer Collection Workspace introduced (DataTable,
 * CollectionToolbar, BaseSelect) -- nothing device-specific lives in
 * those components, only in this view and its own composable/repository.
 */
const router = useRouter()

const { search, status, type, technology, location, locations, sortKey, sortDirection, toggleSort, page, pageSize, devices, total, loading } =
  useDeviceCollection()

const columns: DataTableColumn[] = [
  { key: 'device', label: 'Device', sortable: true },
  { key: 'type', label: 'Type' },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'location', label: 'Location', sortable: true },
  { key: 'assignedCustomer', label: 'Assigned Customer', sortable: true },
]

const statusOptions = [
  { value: 'all', label: 'All Statuses' },
  { value: 'online', label: 'Online' },
  { value: 'offline', label: 'Offline' },
  { value: 'warning', label: 'Warning' },
  { value: 'provisioning', label: 'Provisioning' },
]

const typeOptions = [
  { value: 'all', label: 'All Types' },
  { value: 'ONT', label: 'ONT' },
  { value: 'Router', label: 'Router' },
  { value: 'Switch', label: 'Switch' },
  { value: 'OLT', label: 'OLT' },
]

const technologyOptions = [
  { value: 'any', label: 'Any' },
  { value: 'gpon', label: 'GPON' },
  { value: 'xgs-pon', label: 'XGS-PON' },
]

const locationOptions = computed(() => [
  { value: 'all', label: 'All Locations' },
  ...locations.map((option) => ({ value: option, label: option })),
])

const STATUS_LABELS: Record<DeviceStatus, string> = {
  online: 'Online',
  offline: 'Offline',
  warning: 'Warning',
  provisioning: 'Provisioning',
}

const STATUS_VARIANTS: Record<DeviceStatus, 'success' | 'error' | 'warning' | 'info'> = {
  online: 'success',
  offline: 'error',
  warning: 'warning',
  provisioning: 'info',
}

function rowLabel(device: Device): string {
  return `Open ${device.model}`
}

function openDevice(device: Device) {
  router.push(`/devices/${device.id}`)
}

function handleSort(key: string) {
  toggleSort(key as DeviceSortKey)
}
</script>

<template>
  <div class="device-collection-view">
    <WorkspaceHeader title="Devices" subtitle="Find managed equipment on the live network." />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by device, serial, customer, or location">
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseSelect v-model="type" label="Device Type" :options="typeOptions" />
      <BaseSelect v-model="technology" label="Technology" :options="technologyOptions" />
      <BaseSelect v-model="location" label="Location" :options="locationOptions" />
    </CollectionToolbar>

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="devices"
        :row-key="(device) => device.id"
        :row-label="rowLabel"
        :loading="loading"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No devices match these filters"
        empty-description="Try a different search term or clearing a filter."
        @row-click="openDevice"
        @sort="handleSort"
        @update:page="(next) => (page = next)"
      >
        <template #cell-device="{ row }">
          <div class="device-cell">
            <span class="device-cell__model">{{ row.model }}</span>
            <span class="device-cell__serial">{{ row.serialNumber }}</span>
          </div>
        </template>

        <template #cell-type="{ row }">
          {{ row.type }}
        </template>

        <template #cell-status="{ row }">
          <BaseBadge :variant="STATUS_VARIANTS[row.status as DeviceStatus]">{{
            STATUS_LABELS[row.status as DeviceStatus]
          }}</BaseBadge>
        </template>

        <template #cell-location="{ row }">
          {{ row.location }}
        </template>

        <template #cell-assignedCustomer="{ row }">
          <span v-if="row.assignedCustomerName" class="device-cell__customer">{{ row.assignedCustomerName }}</span>
          <span v-else class="device-cell__unassigned">—</span>
        </template>
      </DataTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.device-collection-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.device-cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.device-cell__model {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.device-cell__serial {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.device-cell__customer {
  color: var(--color-text-primary);
}

.device-cell__unassigned {
  color: var(--color-text-muted);
}
</style>
