<script setup lang="ts">
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { TimelineEvent } from '@/types/timelineEvent'
import { useExplorerActivity } from '@/composables/useExplorerActivity'

/**
 * Explorer's Activity page (docs/09-WORKSPACE-SPECIFICATIONS.md §15): a
 * searchable history of every Event ever recorded (internal/event) --
 * "Added service for Acme Corp," "Retired device test-05," not the
 * generic "Workflow started" text the Dashboard's own Recent Activity
 * widget used to be limited to before richer Events existed. Reached
 * from the sidebar's Explorer dropdown as its second child, and from
 * the Dashboard's Recent Activity "View All" link
 * (DashboardView.vue's `view-all-to="/explorer/activity"`).
 *
 * Row click resolves a destination from the Event's own entityType where
 * one exists as a real route (device/customer/service); every other
 * entityType (contact, location, service_equipment, customer_device)
 * has no Detail View of its own, so those Events carry a `customer_id`
 * in Metadata -- set at write time by the backend handler that already
 * resolved it for the Event's own message -- and route to that
 * Customer's page instead, with no second lookup needed here.
 */
const router = useRouter()
const { search, sortDirection, toggleSort, page, pageSize, events, total, loading } = useExplorerActivity()

const columns: DataTableColumn[] = [
  { key: 'message', label: 'Message' },
  { key: 'type', label: 'Type' },
  { key: 'when', label: 'When', sortable: true },
]

function rowLabel(event: TimelineEvent): string {
  return `Open ${event.message}`
}

function customerIdFromMetadata(event: TimelineEvent): string | null {
  const value = event.metadata?.customer_id
  return typeof value === 'string' ? value : null
}

function openRoute(event: TimelineEvent): string | null {
  switch (event.entityType) {
    case 'device':
      return `/devices/${event.entityId}`
    case 'customer':
      return `/customers/${event.entityId}`
    case 'service':
      return `/services/${event.entityId}`
    default: {
      const customerId = customerIdFromMetadata(event)
      return customerId ? `/customers/${customerId}` : null
    }
  }
}

function openRow(event: TimelineEvent) {
  const route = openRoute(event)
  if (route) router.push(route)
}
</script>

<template>
  <div class="explorer-activity-view">
    <WorkspaceHeader
      title="Activity"
      subtitle="A searchable history of everything that has happened in the OSS."
      :breadcrumbs="[{ label: 'Explorer', to: '/explorer' }]"
    />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by message, type, or entity" />

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="events"
        :row-key="(event) => event.id"
        :row-label="rowLabel"
        :loading="loading"
        sort-key="when"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No activity matches these filters"
        empty-description="Try a different search term."
        @row-click="openRow"
        @sort="toggleSort"
        @update:page="(next) => (page = next)"
      >
        <template #cell-message="{ row }">
          <span class="activity-cell__message">{{ row.message }}</span>
        </template>

        <template #cell-type="{ row }">
          <span class="activity-cell__type">{{ row.type }}</span>
        </template>

        <template #cell-when="{ row }">
          {{ formatDate(row.createdAt) }}
        </template>
      </DataTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.explorer-activity-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.activity-cell__message {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.activity-cell__type {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
