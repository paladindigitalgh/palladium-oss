<script setup lang="ts" generic="TRow">
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import type { IconName } from '@/components/base/BaseIcon.vue'

/**
 * A lightweight table for the small relationship lists that live inside
 * a Detail Workspace section (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 8, "Customer Workspace": Active Services, Assigned Equipment).
 * Distinct from DataTable (components/data-display/DataTable.vue), which
 * is the heavier Collection View building block with sorting and
 * pagination -- a handful of rows inside a SectionCard needs none of
 * that chrome. Shares DataTable's `cell-<columnKey>` scoped-slot
 * convention so the two feel like the same family of component.
 *
 * Non-interactive by default (Customer Detail's Services section: "does
 * not yet need to navigate anywhere"). `clickable` is an explicit opt-in
 * -- added once a second, genuine consumer needed it (Customer Detail's
 * Devices section now opens /devices/:id) -- rather than always wiring
 * click/keyboard handling, which would make every non-clickable use
 * falsely focusable with a handler that does nothing.
 */
export interface SimpleTableColumn {
  key: string
  label: string
}

const props = defineProps<{
  columns: SimpleTableColumn[]
  rows: TRow[]
  rowKey: (row: TRow) => string
  rowLabel?: (row: TRow) => string
  clickable?: boolean
  emptyIcon?: IconName
  emptyTitle: string
  emptyDescription?: string
}>()

const emit = defineEmits<{
  (event: 'row-click', row: TRow): void
}>()

function activate(row: TRow) {
  if (props.clickable) emit('row-click', row)
}

function onRowKeydown(event: KeyboardEvent, row: TRow) {
  if (!props.clickable) return
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    activate(row)
  }
}
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
      <tr
        v-for="row in rows"
        :key="rowKey(row)"
        class="simple-table__row"
        :class="{ 'simple-table__row--clickable': clickable }"
        :tabindex="clickable ? 0 : undefined"
        :aria-label="clickable ? rowLabel?.(row) : undefined"
        @click="activate(row)"
        @keydown="onRowKeydown($event, row)"
      >
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

.simple-table__row--clickable {
  cursor: pointer;
  transition: background-color var(--motion-fast) var(--motion-ease);
}

.simple-table__row--clickable:hover {
  background-color: var(--color-bg);
}

.simple-table__row--clickable:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: -2px;
}

.simple-table__row--clickable:active {
  background-color: var(--color-border);
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
