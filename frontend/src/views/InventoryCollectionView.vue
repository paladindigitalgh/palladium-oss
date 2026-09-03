<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import SiteFormDialog from '@/components/dialogs/SiteFormDialog.vue'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { Site } from '@/types/site'
import { useSiteCollection } from '@/composables/useSiteCollection'

/**
 * The Inventory Collection View: discovery over the Inventory
 * hierarchy's root (Site -> Building -> Room -> Rack -> Device,
 * docs/03-DOMAIN-MODEL.md), built entirely from the same Collection
 * Workspace components NetworkCollectionView.vue introduced -- nothing
 * inventory-specific lives in those components, only in this view and
 * its own composable/repository. Site has no status field, so unlike
 * NetworkCollectionView there is no status filter and only one sortable
 * column (name).
 */
const router = useRouter()

const { search, sortDirection, toggleSort, page, pageSize, sites, total, loading } = useSiteCollection()

const columns: DataTableColumn[] = [
  { key: 'name', label: 'Site', sortable: true },
  { key: 'created', label: 'Created' },
]

function rowLabel(site: Site): string {
  return `Open ${site.name}`
}

function openSite(site: Site) {
  router.push(`/inventory/${site.id}`)
}

const showNewSiteDialog = ref(false)

function handleSiteCreated(site: Site) {
  showNewSiteDialog.value = false
  router.push(`/inventory/${site.id}`)
}
</script>

<template>
  <div class="inventory-collection-view">
    <WorkspaceHeader title="Inventory" subtitle="Search sites, buildings, rooms, and racks.">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showNewSiteDialog = true">New Site</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <SiteFormDialog :open="showNewSiteDialog" @close="showNewSiteDialog = false" @created="handleSiteCreated" />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by name or site ID" />

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="sites"
        :row-key="(site) => site.id"
        :row-label="rowLabel"
        :loading="loading"
        sort-key="name"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No sites match this search"
        empty-description="Try a different search term."
        @row-click="openSite"
        @sort="toggleSort"
        @update:page="(next) => (page = next)"
      >
        <template #cell-name="{ row }">
          <span class="inventory-cell__name">{{ row.name }}</span>
        </template>

        <template #cell-created="{ row }">
          {{ formatDate(row.createdAt) }}
        </template>
      </DataTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.inventory-collection-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.inventory-cell__name {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}
</style>
