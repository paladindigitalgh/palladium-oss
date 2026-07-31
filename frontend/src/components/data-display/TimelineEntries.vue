<script setup lang="ts">
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * Chronological entry list (docs/11-COMPONENT-ARCHITECTURE.md, "Data
 * Display": Timeline). Body content meant to sit inside a `<SectionCard
 * title="Timeline">` -- SectionCard now provides the card chrome and
 * heading, so unlike the old TimelinePanel this component owns neither.
 * Generic over entry shape: it renders whatever {label, timestamp,
 * description} entries it is given and has no idea what generated them
 * (docs/02-DESIGN-PRINCIPLES.md principle 9, "Events as Operational
 * History").
 */
withDefaults(
  defineProps<{
    entries?: { id: string; label: string; timestamp?: string; description?: string }[]
  }>(),
  { entries: () => [] },
)
</script>

<template>
  <BaseEmptyState
    v-if="entries.length === 0"
    icon="clock"
    title="No activity yet"
    description="Events will appear here as they happen."
  />
  <ol v-else class="timeline-entries">
    <li v-for="entry in entries" :key="entry.id" class="timeline-entries__entry">
      <BaseIcon name="clock" size="sm" class="timeline-entries__icon" />
      <div>
        <p class="timeline-entries__label">{{ entry.label }}</p>
        <p v-if="entry.timestamp" class="timeline-entries__timestamp">{{ entry.timestamp }}</p>
        <p v-if="entry.description" class="timeline-entries__description">
          {{ entry.description }}
        </p>
      </div>
    </li>
  </ol>
</template>

<style scoped>
.timeline-entries {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.timeline-entries__entry {
  display: flex;
  gap: var(--space-3);
}

.timeline-entries__icon {
  color: var(--color-text-muted);
  margin-top: 2px;
}

.timeline-entries__label {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.timeline-entries__timestamp {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.timeline-entries__description {
  margin-top: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}
</style>
