<script setup lang="ts">
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import type { Note } from '@/types/note'

/**
 * Read-only operator notes -- extracted once Device Detail's Notes
 * section needed the exact list Customer Detail's Notes section already
 * rendered inline.
 */
withDefaults(
  defineProps<{
    notes?: Note[]
  }>(),
  { notes: () => [] },
)
</script>

<template>
  <BaseEmptyState
    v-if="notes.length === 0"
    icon="check"
    title="No notes yet"
    description="Operator notes will appear here."
  />
  <ul v-else class="notes-list">
    <li v-for="note in notes" :key="note.id" class="notes-list__item">
      <div class="notes-list__meta">
        <span class="notes-list__author">{{ note.author }}</span>
        <span class="notes-list__timestamp">{{ note.timestamp }}</span>
      </div>
      <p class="notes-list__body">{{ note.body }}</p>
    </li>
  </ul>
</template>

<style scoped>
.notes-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.notes-list__item {
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.notes-list__item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.notes-list__meta {
  display: flex;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-1);
}

.notes-list__author {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.notes-list__timestamp {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.notes-list__body {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}
</style>
