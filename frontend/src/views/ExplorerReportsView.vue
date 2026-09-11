<script setup lang="ts">
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable from '@/components/data-display/DataTable.vue'
import { useExplorerReports, type ReportRecord } from '@/composables/useExplorerReports'

/**
 * Explorer's Reports page (docs/09-WORKSPACE-SPECIFICATIONS.md §15):
 * Palladium's curated set of cross-domain reports, each backed by a real
 * internal/report SQL join, not a dynamic ad hoc query builder (see
 * useExplorerReports.ts's own doc comment on that decision and on why
 * this view is not shaped like DeviceCollectionView.vue's
 * refetch-per-filter pattern). All state and orchestration live in
 * useExplorerReports -- this view is markup only. Reached from the
 * sidebar's Explorer dropdown as its first child (Explorer itself has no
 * page of its own, see router/index.ts's /explorer redirect); its
 * sibling, the Activity page (ExplorerActivityView.vue), is a separate
 * page entirely, not a tile within this one.
 *
 * No report runs automatically on landing (2026-09-11, user's explicit
 * request): the report tiles are the landing content, and clicking one
 * both runs it and stays visible as the switcher for running a
 * different report -- the same row of tiles serves both jobs rather
 * than a picker tucked in a corner.
 */
const router = useRouter()
const {
  reports,
  selectedReportId,
  selectedReport,
  selectReport,
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
} = useExplorerReports()

function rowLabel(row: ReportRecord): string {
  return `Open ${row.customerName ?? row.deviceName ?? 'record'}`
}

function openRow(row: ReportRecord) {
  router.push(openRoute(row))
}
</script>

<template>
  <div class="explorer-view">
    <WorkspaceHeader
      title="Reports"
      subtitle="Pull data out of the OSS: pick a report, filter it, and export it to CSV."
      :breadcrumbs="[{ label: 'Explorer', to: '/explorer' }]"
    >
      <template v-if="selectedReport" #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" :disabled="loading || total === 0" @click="exportCsv">
              Export CSV
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <div class="report-picker">
      <button
        v-for="report in reports"
        :key="report.id"
        type="button"
        class="report-tile"
        :class="{ 'report-tile--active': report.id === selectedReportId }"
        @click="selectReport(report.id)"
      >
        <span class="report-tile__label">{{ report.label }}</span>
        <span class="report-tile__description">{{ report.description }}</span>
      </button>
    </div>

    <template v-if="selectedReport">
      <CollectionToolbar v-model:search="search" search-placeholder="Search this report" />

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
    </template>

    <BaseCard v-else :padded="false">
      <BaseEmptyState
        icon="explorer"
        title="Select a report above to get started"
        description="Each report is fetched fresh and can be searched, sorted, and exported to CSV."
      />
    </BaseCard>
  </div>
</template>

<style scoped>
.explorer-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.report-picker {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}

.report-tile {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-1);
  flex: 1 1 240px;
  min-width: 200px;
  padding: var(--space-4);
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-align: left;
  cursor: pointer;
  font: inherit;
  color: inherit;
  transition: border-color 0.15s ease, background-color 0.15s ease;
}

.report-tile:hover {
  border-color: var(--color-brand);
}

.report-tile--active {
  border-color: var(--color-brand);
  background-color: var(--color-info-bg);
}

.report-tile__label {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.report-tile--active .report-tile__label {
  color: var(--color-brand);
}

.report-tile__description {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}
</style>
