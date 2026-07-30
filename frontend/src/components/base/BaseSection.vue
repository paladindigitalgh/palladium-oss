<script setup lang="ts">
import BaseCard from './BaseCard.vue'

/**
 * A titled content grouping, built on BaseCard. Distinct from
 * WorkspaceContent (components/workspace/WorkspaceContent.vue):
 * WorkspaceContent is the layout *region* a Workspace reserves for its
 * primary content; BaseSection is a reusable grouped-content block that
 * can appear inside that region (or anywhere else) any number of times.
 */
defineProps<{
  title?: string
  description?: string
}>()
</script>

<template>
  <BaseCard class="base-section">
    <div v-if="title || $slots.actions" class="base-section__header">
      <div>
        <h2 v-if="title" class="base-section__title">{{ title }}</h2>
        <p v-if="description" class="base-section__description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="base-section__actions">
        <slot name="actions" />
      </div>
    </div>
    <div class="base-section__body">
      <slot />
    </div>
  </BaseCard>
</template>

<style scoped>
.base-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.base-section__title {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.base-section__description {
  margin-top: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}
</style>
