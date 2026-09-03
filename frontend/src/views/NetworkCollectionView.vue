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
import AccessNetworkFormDialog from '@/components/dialogs/AccessNetworkFormDialog.vue'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { AccessNetwork, AccessNetworkStatus } from '@/types/accessNetwork'
import { useAccessNetworkCollection, type AccessNetworkSortKey } from '@/composables/useAccessNetworkCollection'

/**
 * The Network Collection View: discovery over the access-network
 * hierarchy's root (AccessNetwork -> OLT -> PONPort -> AccessInterface ->
 * AccessAttachment, docs/03-DOMAIN-MODEL.md), built entirely from the
 * same Collection Workspace components DeviceCollectionView.vue
 * introduced -- nothing network-specific lives in those components, only
 * in this view and its own composable/repository.
 */
const router = useRouter()

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

function rowLabel(accessNetwork: AccessNetwork): string {
  return `Open ${accessNetwork.name}`
}

function openAccessNetwork(accessNetwork: AccessNetwork) {
  router.push(`/network/${accessNetwork.id}`)
}

function handleSort(key: string) {
  toggleSort(key as AccessNetworkSortKey)
}

const showNewAccessNetworkDialog = ref(false)

function handleAccessNetworkCreated(accessNetwork: AccessNetwork) {
  showNewAccessNetworkDialog.value = false
  router.push(`/network/${accessNetwork.id}`)
}
</script>

<template>
  <div class="network-collection-view">
    <WorkspaceHeader title="Network" subtitle="Search access networks, OLTs, and PON ports.">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showNewAccessNetworkDialog = true">New Access Network</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <AccessNetworkFormDialog
      :open="showNewAccessNetworkDialog"
      @close="showNewAccessNetworkDialog = false"
      @created="handleAccessNetworkCreated"
    />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by name or access network ID">
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
    </CollectionToolbar>

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="accessNetworks"
        :row-key="(accessNetwork) => accessNetwork.id"
        :row-label="rowLabel"
        :loading="loading"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No access networks match these filters"
        empty-description="Try a different search term or clearing a filter."
        @row-click="openAccessNetwork"
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
