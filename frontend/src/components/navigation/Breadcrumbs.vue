<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'

/**
 * The shell-level "where am I" trail (this milestone's goal 6: "Top
 * Navigation... Breadcrumb area"), showing which primary navigation
 * section is active. Distinct from WorkspaceHeader's breadcrumbs prop
 * (components/workspace/WorkspaceHeader.vue), which is for a future
 * domain workspace's entity-relationship trail (e.g. Customer > Service)
 * -- see that component's doc comment for the full reasoning. Generic
 * over items so it stays reusable regardless of which trail it renders.
 */
defineProps<{
  items: { label: string; to?: RouteLocationRaw }[]
}>()
</script>

<template>
  <nav class="breadcrumbs" aria-label="Breadcrumb">
    <template v-for="(item, index) in items" :key="item.label">
      <RouterLink v-if="item.to" :to="item.to" class="breadcrumbs__item">{{
        item.label
      }}</RouterLink>
      <span v-else class="breadcrumbs__item breadcrumbs__item--current">{{ item.label }}</span>
      <span v-if="index < items.length - 1" class="breadcrumbs__separator">/</span>
    </template>
  </nav>
</template>

<style scoped>
.breadcrumbs {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  min-width: 0;
}

.breadcrumbs__item {
  color: var(--color-text-muted);
  text-decoration: none;
  white-space: nowrap;
}

.breadcrumbs__item:hover {
  color: var(--color-text-primary);
}

.breadcrumbs__item--current {
  color: var(--color-text-primary);
  font-weight: var(--font-weight-medium);
}

.breadcrumbs__separator {
  color: var(--color-text-muted);
}
</style>
