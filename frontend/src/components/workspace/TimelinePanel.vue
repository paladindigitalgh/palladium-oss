<script setup lang="ts">
import BaseCard from '@/components/base/BaseCard.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * Chronological history region (docs/11-COMPONENT-ARCHITECTURE.md:
 * "Timeline Panel: Provisioning, Configuration, Audit events, Inventory
 * changes, Plugin actions"). Generic over entry shape -- it renders
 * whatever {label, timestamp, description} entries it's given and has
 * no knowledge of what generated them, so the same component serves
 * every future domain's activity history (docs/02-DESIGN-PRINCIPLES.md
 * principle 8, "Events as Operational History").
 */
withDefaults(
  defineProps<{
    title?: string
    entries?: { id: string; label: string; timestamp?: string; description?: string }[]
  }>(),
  { title: 'Timeline', entries: () => [] },
)
</script>

<template>
  <BaseCard class="timeline-panel">
    <h2 class="timeline-panel__title">{{ title }}</h2>
    <BaseEmptyState
      v-if="entries.length === 0"
      icon="clock"
      title="No activity yet"
      description="Events for this workspace will appear here as they happen."
    />
    <ol v-else class="timeline-panel__list">
      <li v-for="entry in entries" :key="entry.id" class="timeline-panel__entry">
        <BaseIcon name="clock" size="sm" class="timeline-panel__icon" />
        <div>
          <p class="timeline-panel__label">{{ entry.label }}</p>
          <p v-if="entry.timestamp" class="timeline-panel__timestamp">{{ entry.timestamp }}</p>
          <p v-if="entry.description" class="timeline-panel__description">
            {{ entry.description }}
          </p>
        </div>
      </li>
    </ol>
  </BaseCard>
</template>

<style scoped>
.timeline-panel__title {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  margin-bottom: var(--space-3);
}

.timeline-panel__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.timeline-panel__entry {
  display: flex;
  gap: var(--space-3);
}

.timeline-panel__icon {
  color: var(--color-text-muted);
  margin-top: 2px;
}

.timeline-panel__label {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.timeline-panel__timestamp {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.timeline-panel__description {
  margin-top: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}
</style>
