<script setup lang="ts" generic="TRow">
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'

/**
 * The reusable table every Collection View is built from
 * (docs/08-DESIGN-SYSTEM.md section 15, "Tables";
 * docs/11-COMPONENT-ARCHITECTURE.md, "Tables are for collections, cards
 * are for individual objects"). Generic over its row type and driven
 * entirely by props/events/slots -- it has no notion of Customers,
 * Devices, or any other business object, so the same component serves
 * every future Collection Workspace.
 *
 * Cell content is supplied per column via a `cell-<columnKey>` scoped
 * slot; the same slot is reused for both the desktop table and the
 * mobile card layout below 960px (docs/08-DESIGN-SYSTEM.md section 24),
 * so a consumer never writes cell markup twice. Sorting, pagination, and
 * the empty/loading states are driven by props and reported back via
 * events -- this component owns no server data itself
 * (docs/11-COMPONENT-ARCHITECTURE.md, "Separate business logic from
 * presentation").
 *
 * Bulk selection and column visibility are documented future
 * extensions, not built here -- this milestone only needs sorting,
 * pagination, loading/empty states, row click, and keyboard access.
 */
export interface DataTableColumn {
  key: string
  label: string
  sortable?: boolean
}

const props = defineProps<{
  columns: DataTableColumn[]
  rows: TRow[]
  rowKey: (row: TRow) => string
  rowLabel?: (row: TRow) => string
  loading?: boolean
  sortKey?: string
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
  total?: number
  emptyTitle?: string
  emptyDescription?: string
}>()

const emit = defineEmits<{
  (event: 'row-click', row: TRow): void
  (event: 'sort', key: string): void
  (event: 'update:page', page: number): void
}>()

function ariaSortFor(column: DataTableColumn): 'ascending' | 'descending' | 'none' {
  if (!column.sortable || column.key !== props.sortKey) return 'none'
  return props.sortDirection === 'desc' ? 'descending' : 'ascending'
}

function activate(row: TRow) {
  emit('row-click', row)
}

function onRowKeydown(event: KeyboardEvent, row: TRow) {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    activate(row)
  }
}

const showPagination = () => props.total !== undefined && props.pageSize !== undefined

function totalPages(): number {
  if (props.total === undefined || props.pageSize === undefined) return 1
  return Math.max(1, Math.ceil(props.total / props.pageSize))
}

function rangeLabel(): string {
  if (props.total === undefined || props.pageSize === undefined || props.page === undefined) return ''
  if (props.total === 0) return 'No results'
  const start = (props.page - 1) * props.pageSize + 1
  const end = Math.min(props.page * props.pageSize, props.total)
  return `Showing ${start}–${end} of ${props.total}`
}

function goToPage(next: number) {
  const clamped = Math.min(Math.max(next, 1), totalPages())
  if (clamped !== props.page) emit('update:page', clamped)
}
</script>

<template>
  <div class="data-table">
    <div v-if="loading" class="data-table__status">
      <BaseLoadingState :lines="6" />
    </div>

    <div v-else-if="rows.length === 0" class="data-table__status">
      <slot name="empty">
        <BaseEmptyState
          icon="search"
          :title="emptyTitle ?? 'No results found'"
          :description="emptyDescription ?? 'Try adjusting your search or filters.'"
        />
      </slot>
    </div>

    <template v-else>
      <table class="data-table__table">
        <thead>
          <tr>
            <th
              v-for="column in columns"
              :key="column.key"
              scope="col"
              class="data-table__header-cell"
              :aria-sort="ariaSortFor(column)"
            >
              <button
                v-if="column.sortable"
                type="button"
                class="data-table__sort-button"
                @click="emit('sort', column.key)"
              >
                {{ column.label }}
                <BaseIcon
                  v-if="column.key === sortKey"
                  name="chevron-down"
                  size="sm"
                  class="data-table__sort-icon"
                  :class="{ 'data-table__sort-icon--asc': sortDirection === 'asc' }"
                />
                <BaseIcon v-else name="sort" size="sm" class="data-table__sort-icon data-table__sort-icon--neutral" />
              </button>
              <span v-else>{{ column.label }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in rows"
            :key="rowKey(row)"
            class="data-table__row"
            tabindex="0"
            :aria-label="rowLabel?.(row)"
            @click="activate(row)"
            @keydown="onRowKeydown($event, row)"
          >
            <td v-for="column in columns" :key="column.key" class="data-table__cell">
              <slot :name="`cell-${column.key}`" :row="row" />
            </td>
          </tr>
        </tbody>
      </table>

      <ul class="data-table__cards">
        <li
          v-for="row in rows"
          :key="rowKey(row)"
          class="data-table__card"
          tabindex="0"
          :aria-label="rowLabel?.(row)"
          @click="activate(row)"
          @keydown="onRowKeydown($event, row)"
        >
          <div class="data-table__card-title">
            <slot :name="`cell-${columns[0].key}`" :row="row" />
          </div>
          <div v-for="column in columns.slice(1)" :key="column.key" class="data-table__card-field">
            <span class="data-table__card-label">{{ column.label }}</span>
            <span class="data-table__card-value"><slot :name="`cell-${column.key}`" :row="row" /></span>
          </div>
        </li>
      </ul>

      <div v-if="showPagination()" class="data-table__pagination">
        <span class="data-table__range">{{ rangeLabel() }}</span>
        <div class="data-table__pagination-controls">
          <BaseButton
            variant="ghost"
            size="sm"
            :disabled="(page ?? 1) <= 1"
            @click="goToPage((page ?? 1) - 1)"
          >
            Previous
          </BaseButton>
          <span class="data-table__page-indicator">Page {{ page ?? 1 }} of {{ totalPages() }}</span>
          <BaseButton
            variant="ghost"
            size="sm"
            :disabled="(page ?? 1) >= totalPages()"
            @click="goToPage((page ?? 1) + 1)"
          >
            Next
          </BaseButton>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.data-table {
  display: flex;
  flex-direction: column;
}

.data-table__status {
  padding: var(--space-5);
}

.data-table__table {
  width: 100%;
  border-collapse: collapse;
}

.data-table__header-cell {
  text-align: left;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.data-table__sort-button {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  background: transparent;
  border: none;
  padding: 0;
  font: inherit;
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  cursor: pointer;
  border-radius: var(--radius-sm);
}

.data-table__sort-button:hover {
  color: var(--color-text-primary);
}

.data-table__header-cell > span {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.data-table__sort-icon {
  color: var(--color-text-muted);
  transition: transform var(--motion-fast) var(--motion-ease);
}

.data-table__sort-icon--asc {
  transform: rotate(180deg);
}

.data-table__sort-icon--neutral {
  opacity: 0.5;
}

.data-table__row {
  cursor: pointer;
  transition: background-color var(--motion-fast) var(--motion-ease);
}

.data-table__row:hover {
  background-color: var(--color-bg);
}

.data-table__row:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: -2px;
}

.data-table__row:active {
  background-color: var(--color-border);
}

.data-table__row:not(:last-child) {
  border-bottom: 1px solid var(--color-border);
}

.data-table__cell {
  padding: var(--space-3) var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  vertical-align: middle;
}

.data-table__cards {
  display: none;
}

.data-table__pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-4);
  border-top: 1px solid var(--color-border);
  flex-wrap: wrap;
}

.data-table__range {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.data-table__pagination-controls {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.data-table__page-indicator {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  white-space: nowrap;
}

@media (max-width: 960px) {
  .data-table__table {
    display: none;
  }

  .data-table__cards {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-4);
  }

  .data-table__card {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: background-color var(--motion-fast) var(--motion-ease);
  }

  .data-table__card:hover {
    background-color: var(--color-bg);
  }

  .data-table__card:focus-visible {
    outline: 2px solid var(--color-brand);
    outline-offset: 2px;
  }

  .data-table__card:active {
    background-color: var(--color-border);
  }

  .data-table__card-title {
    font-size: var(--font-size-md);
    font-weight: var(--font-weight-semibold);
    color: var(--color-text-primary);
  }

  .data-table__card-field {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-3);
    font-size: var(--font-size-sm);
  }

  .data-table__card-label {
    color: var(--color-text-muted);
  }

  .data-table__card-value {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    color: var(--color-text-primary);
    text-align: right;
  }

  .data-table__pagination {
    justify-content: center;
    text-align: center;
  }
}
</style>
