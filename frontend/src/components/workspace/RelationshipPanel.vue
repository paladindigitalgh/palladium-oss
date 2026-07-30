<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * Displays related entities (docs/11-COMPONENT-ARCHITECTURE.md:
 * "Relationship Panel... Relationships should always be navigable").
 * Deliberately generic: it has no notion of the Customer -> Service ->
 * ONU -> PON -> OLT -> Cabinet -> Site chain the docs use as an example
 * -- it only knows a list of {label, description, to} items, so it can
 * front any relationship a future domain workspace needs. `to` is
 * optional because no domain routes exist yet in this milestone; when
 * absent the row renders as a non-interactive placeholder.
 */
withDefaults(
  defineProps<{
    title?: string
    items?: { id: string; label: string; description?: string; to?: RouteLocationRaw }[]
  }>(),
  { title: 'Related resources', items: () => [] },
)
</script>

<template>
  <BaseCard class="relationship-panel">
    <h2 class="relationship-panel__title">{{ title }}</h2>
    <BaseEmptyState
      v-if="items.length === 0"
      icon="network"
      title="No related resources yet"
      description="Related resources will appear here once this workspace is connected to real data."
    />
    <ul v-else class="relationship-panel__list">
      <li v-for="item in items" :key="item.id">
        <component
          :is="item.to ? RouterLink : 'div'"
          :to="item.to"
          class="relationship-panel__item"
        >
          <div>
            <p class="relationship-panel__label">{{ item.label }}</p>
            <p v-if="item.description" class="relationship-panel__description">
              {{ item.description }}
            </p>
          </div>
          <BaseIcon v-if="item.to" name="arrow-right" size="sm" />
        </component>
      </li>
    </ul>
  </BaseCard>
</template>

<style scoped>
.relationship-panel__title {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  margin-bottom: var(--space-3);
}

.relationship-panel__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.relationship-panel__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  color: inherit;
  text-decoration: none;
}

a.relationship-panel__item:hover {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.relationship-panel__label {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.relationship-panel__description {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
