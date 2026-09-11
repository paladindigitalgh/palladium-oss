<script setup lang="ts">
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable from '@/components/data-display/DataTable.vue'
import { useExplorer, type ReportRecord } from '@/composables/useExplorer'

/**
 * Explorer (docs/09-WORKSPACE-SPECIFICATIONS.md §15): Palladium's
 * curated set of cross-domain reports, each backed by a real
 * internal/report SQL join, not a dynamic ad hoc query builder (see
 * useExplorer.ts's own doc comment on that decision and on why this
 * view is not shaped like DeviceCollectionView.vue's refetch-per-filter
 * pattern). All state and orchestration live in useExplorer -- this
 * view is markup only.
 */
const router = useRouter()
const {
  reportOptions,
  selectedReportId,
  selectedReport,
  search,
  sortKey,
  sortDirection,
  toggleSort,
  page,
  pageSize,
  pageRows,
  total,
  loading,
  openRoute,
  exportCsv,
} = useExplorer()

function rowLabel(row: ReportRecord): string {
  return `Open ${row.customerName ?? row.deviceName ?? 'record'}`
}

function openRow(row: ReportRecord) {
  router.push(openRoute(row))
}
</script>

<template>
  <div class="explorer-view">
    <WorkspaceHeader title="Explorer" subtitle="Pull data out of the OSS: browse a report, filter it, and export it to CSV.">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" :disabled="loading || total === 0" @click="exportCsv">
              Export CSV
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <CollectionToolbar v-model:search="search" search-placeholder="Search this report">
      <BaseSelect v-model="selectedReportId" label="Report" :options="reportOptions" />
    </CollectionToolbar>

    <BaseCard :padded="false">
      <DataTable
        :columns="selectedReport.columns"
        :rows="pageRows"
        :row-key="(row) => row.key"
        :row-label="rowLabel"
        :loading="loading"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No results match these filters"
        empty-description="Try a different search term or a different report."
        @row-click="openRow"
        @sort="toggleSort"
        @update:page="(next) => (page = next)"
      >
        <template v-for="column in selectedReport.columns" :key="column.key" #[`cell-${column.key}`]="{ row }">
          {{ row[column.key] }}
        </template>
      </DataTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.explorer-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
</style>
