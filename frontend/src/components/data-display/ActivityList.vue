<script setup lang="ts">
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'

/**
 * A compact label + timestamp list -- lighter weight than TimelineEntries
 * (components/data-display/TimelineEntries.vue), which also renders an
 * icon and an optional description for a longer historical record.
 * Extracted from DashboardView's "Recent Activity" widget once the
 * Customer Detail Workspace needed the identical shape for its own
 * Recent Activity section (docs/11-COMPONENT-ARCHITECTURE.md, "Future
 * Evolution": promote a pattern once a second place actually needs it).
 */
withDefaults(
  defineProps<{
    entries?: readonly { id: string; label: string; timestamp?: string }[]
  }>(),
  { entries: () => [] },
)
</script>

<template>
  <BaseEmptyState
    v-if="entries.length === 0"
    icon="clock"
    title="No recent activity"
    description="Operational events will appear here as they happen."
  />
  <ol v-else class="activity-list">
    <li v-for="entry in entries" :key="entry.id" class="activity-list__item">
      <span class="activity-list__label">{{ entry.label }}</span>
      <span v-if="entry.timestamp" class="activity-list__timestamp">{{ entry.timestamp }}</span>
    </li>
  </ol>
</template>

<style scoped>
.activity-list {
  display: flex;
  flex-direction: column;
}

.activity-list__item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}

.activity-list__item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.activity-list__label {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.activity-list__timestamp {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  white-space: nowrap;
}
</style>
