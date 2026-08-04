<script setup lang="ts" generic="TRow">
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import type { IconName } from '@/components/base/BaseIcon.vue'

/**
 * A lightweight, non-interactive table for the small relationship lists
 * that live inside a Detail Workspace section
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, section 8, "Customer Workspace":
 * Active Services, Assigned Equipment). Distinct from DataTable
 * (components/data-display/DataTable.vue), which is the heavier
 * Collection View building block with sorting, pagination, and row
 * navigation -- a handful of rows inside a SectionCard needs none of
 * that chrome. Shares DataTable's `cell-<columnKey>` scoped-slot
 * convention so the two feel like the same family of component.
 */
export interface SimpleTableColumn {
  key: string
  label: string
}

defineProps<{
  columns: SimpleTableColumn[]
  rows: TRow[]
  rowKey: (row: TRow) => string
  emptyIcon?: IconName
  emptyTitle: string
  emptyDescription?: string
}>()
</script>

<template>
  <BaseEmptyState v-if="rows.length === 0" :icon="emptyIcon" :title="emptyTitle" :description="emptyDescription" />
  <table v-else class="simple-table">
    <thead>
      <tr>
        <th v-for="column in columns" :key="column.key" scope="col" class="simple-table__header-cell">
          {{ column.label }}
        </th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="row in rows" :key="rowKey(row)" class="simple-table__row">
        <td v-for="column in columns" :key="column.key" class="simple-table__cell">
          <slot :name="`cell-${column.key}`" :row="row" />
        </td>
      </tr>
    </tbody>
  </table>
</template>

<style scoped>
.simple-table {
  width: 100%;
  border-collapse: collapse;
}

.simple-table__header-cell {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid var(--color-border);
}

.simple-table__row:not(:last-child) {
  border-bottom: 1px solid var(--color-border);
}

.simple-table__cell {
  padding: var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  vertical-align: middle;
}

@media (max-width: 640px) {
  .simple-table {
    display: block;
    overflow-x: auto;
  }
}
</style>
