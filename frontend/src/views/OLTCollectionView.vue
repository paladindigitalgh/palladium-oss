<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import OLTFormDialog from '@/components/dialogs/OLTFormDialog.vue'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { OLT } from '@/types/olt'
import { useOLTCollection } from '@/composables/useOLTCollection'

/**
 * The Network Collection View: discovery over the Network hierarchy's
 * root (OLT -> PON Port -> Access Interface -> Access Attachment,
 * docs/03-DOMAIN-MODEL.md), built entirely from the same Collection
 * Workspace components DeviceCollectionView.vue introduced -- nothing
 * network-specific lives in those components, only in this view and its
 * own composable/repository. OLT is a top-level collection: Palladium
 * only ever manages a single physical network per instance, so there is
 * no grouping entity above it to click through first.
 */
const router = useRouter()

const { search, sortDirection, toggleSort, page, pageSize, olts, total, loading } = useOLTCollection()

const columns: DataTableColumn[] = [
  { key: 'name', label: 'OLT', sortable: true },
  { key: 'managementIpAddress', label: 'Management IP' },
  { key: 'created', label: 'Created' },
]

function rowLabel(olt: OLT): string {
  return `Open ${olt.name}`
}

function openOLT(olt: OLT) {
  router.push(`/network/olts/${olt.id}`)
}

const showNewOLTDialog = ref(false)

function handleOLTCreated(olt: OLT) {
  showNewOLTDialog.value = false
  router.push(`/network/olts/${olt.id}`)
}
</script>

<template>
  <div class="network-collection-view">
    <WorkspaceHeader title="Network" subtitle="Search OLTs, PON ports, and access interfaces.">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showNewOLTDialog = true">New OLT</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <OLTFormDialog :open="showNewOLTDialog" @close="showNewOLTDialog = false" @created="handleOLTCreated" />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by name, management IP, or OLT ID" />

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="olts"
        :row-key="(olt) => olt.id"
        :row-label="rowLabel"
        :loading="loading"
        sort-key="name"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No OLTs match these filters"
        empty-description="Try a different search term, or add the first OLT."
        @row-click="openOLT"
        @sort="toggleSort"
        @update:page="(next) => (page = next)"
      >
        <template #cell-name="{ row }">
          <span class="network-cell__name">{{ row.name }}</span>
        </template>

        <template #cell-managementIpAddress="{ row }">
          {{ row.managementIpAddress || '—' }}
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
