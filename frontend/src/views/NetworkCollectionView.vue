<script setup lang="ts">
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { AccessNetworkStatus } from '@/types/accessNetwork'
import { useAccessNetworkCollection, type AccessNetworkSortKey } from '@/composables/useAccessNetworkCollection'

/**
 * The Network Collection View: discovery over the access-network
 * hierarchy's root (AccessNetwork -> OLT -> PONPort -> AccessInterface ->
 * AccessAttachment, docs/03-DOMAIN-MODEL.md), built entirely from the
 * same Collection Workspace components DeviceCollectionView.vue
 * introduced -- nothing network-specific lives in those components, only
 * in this view and its own composable/repository. Create and row-click
 * navigation land in the next commit alongside AccessNetworkDetailView.vue
 * -- there is nowhere for a row click to go yet.
 */
const { search, status, sortKey, sortDirection, toggleSort, page, pageSize, accessNetworks, total, loading } =
  useAccessNetworkCollection()

const columns: DataTableColumn[] = [
  { key: 'name', label: 'Access Network', sortable: true },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'created', label: 'Created' },
]

const statusOptions = [
  { value: 'all', label: 'All Statuses' },
  { value: 'Active', label: 'Active' },
  { value: 'Inactive', label: 'Inactive' },
]

const STATUS_VARIANTS: Record<AccessNetworkStatus, 'success' | 'neutral'> = {
  Active: 'success',
  Inactive: 'neutral',
}

function handleSort(key: string) {
  toggleSort(key as AccessNetworkSortKey)
}
</script>

<template>
  <div class="network-collection-view">
    <WorkspaceHeader title="Network" subtitle="Search access networks, OLTs, and PON ports." />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by name or access network ID">
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
    </CollectionToolbar>

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="accessNetworks"
        :row-key="(accessNetwork) => accessNetwork.id"
        :loading="loading"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No access networks match these filters"
        empty-description="Try a different search term or clearing a filter."
        @sort="handleSort"
        @update:page="(next) => (page = next)"
      >
        <template #cell-name="{ row }">
          <span class="network-cell__name">{{ row.name }}</span>
        </template>

        <template #cell-status="{ row }">
          <BaseBadge :variant="STATUS_VARIANTS[row.status as AccessNetworkStatus]">{{ row.status }}</BaseBadge>
        </template>

        <template #cell-created="{ row }">
          {{ formatDate(row.createdAt) }}
        </template>
      </DataTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.network-collection-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.network-cell__name {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}
</style>
