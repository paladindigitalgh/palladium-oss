<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import DeviceFormDialog from '@/components/dialogs/DeviceFormDialog.vue'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { Device, DeviceStatus } from '@/types/device'
import { useDeviceCollection, type DeviceSortKey } from '@/composables/useDeviceCollection'

/**
 * The Device Collection View (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * "Collection View & Detail View"; section 10, "Device Workspace"):
 * discovery over Palladium's physical inventory, backed by the real
 * Device API. Built entirely from the same Collection Workspace
 * components the Customer Collection Workspace introduced (DataTable,
 * CollectionToolbar, BaseSelect) -- nothing device-specific lives in
 * those components, only in this view and its own composable/repository.
 */
const router = useRouter()

const { search, status, sortKey, sortDirection, toggleSort, page, pageSize, devices, total, loading } = useDeviceCollection()

const columns: DataTableColumn[] = [
  { key: 'device', label: 'Device', sortable: true },
  { key: 'manufacturer', label: 'Manufacturer / Model' },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'created', label: 'Created' },
]

const statusOptions = [
  { value: 'all', label: 'All Statuses' },
  { value: 'Ordered', label: 'Ordered' },
  { value: 'Received', label: 'Received' },
  { value: 'InStock', label: 'In Stock' },
  { value: 'Installed', label: 'Installed' },
  { value: 'Maintenance', label: 'Maintenance' },
  { value: 'Retired', label: 'Retired' },
  { value: 'Disposed', label: 'Disposed' },
]

const STATUS_VARIANTS: Record<DeviceStatus, 'success' | 'error' | 'warning' | 'info' | 'neutral'> = {
  Ordered: 'info',
  Received: 'info',
  InStock: 'success',
  Installed: 'success',
  Maintenance: 'warning',
  Retired: 'neutral',
  Disposed: 'neutral',
}

function rowLabel(device: Device): string {
  return `Open ${device.name}`
}

function openDevice(device: Device) {
  router.push(`/devices/${device.id}`)
}

function handleSort(key: string) {
  toggleSort(key as DeviceSortKey)
}

const showNewDeviceDialog = ref(false)

function handleDeviceCreated(device: Device) {
  showNewDeviceDialog.value = false
  router.push(`/devices/${device.id}`)
}
</script>

<template>
  <div class="device-collection-view">
    <WorkspaceHeader title="Devices" subtitle="Search, filter, and open a device workspace.">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showNewDeviceDialog = true">New Device</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <DeviceFormDialog :open="showNewDeviceDialog" @close="showNewDeviceDialog = false" @created="handleDeviceCreated" />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by name, serial, manufacturer, or model">
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
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
            <span class="device-cell__name">{{ row.name }}</span>
            <span class="device-cell__serial">{{ row.serialNumber }}</span>
          </div>
        </template>

        <template #cell-manufacturer="{ row }"> {{ row.manufacturer }} {{ row.model }} </template>

        <template #cell-status="{ row }">
          <BaseBadge :variant="STATUS_VARIANTS[row.status as DeviceStatus]">{{ row.status }}</BaseBadge>
        </template>

        <template #cell-created="{ row }">
          {{ formatDate(row.createdAt) }}
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

.device-cell__name {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.device-cell__serial {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
