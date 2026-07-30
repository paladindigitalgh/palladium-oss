<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import BaseBadge from '@/components/base/BaseBadge.vue'

/**
 * The entity-identity region of a Workspace (docs/11-COMPONENT-
 * ARCHITECTURE.md, "Workspace Header": object name, status, breadcrumbs,
 * primary actions). Generic across every future domain -- it only knows
 * about a title, an optional status badge, and breadcrumbs, never about
 * what kind of entity it is heading.
 *
 * Primary actions render through the `actions` slot (typically a
 * WorkspaceActions instance) rather than a dedicated prop: the component
 * tree in docs/11-COMPONENT-ARCHITECTURE.md lists WorkspaceActions as a
 * sibling of WorkspaceHeader, but its own prose places actions inside the
 * header region. Slotting WorkspaceActions into WorkspaceHeader satisfies
 * both: WorkspaceActions stays an independent, reusable component, and it
 * still renders where the header visually needs it.
 */
defineProps<{
  title: string
  subtitle?: string
  status?: { label: string; variant?: 'success' | 'warning' | 'error' | 'info' | 'neutral' }
  breadcrumbs?: { label: string; to?: RouteLocationRaw }[]
}>()
</script>

<template>
  <header class="workspace-header">
    <nav v-if="breadcrumbs?.length" class="workspace-header__breadcrumbs" aria-label="Breadcrumb">
      <template v-for="(crumb, index) in breadcrumbs" :key="crumb.label">
        <RouterLink v-if="crumb.to" :to="crumb.to" class="workspace-header__crumb">{{
          crumb.label
        }}</RouterLink>
        <span v-else class="workspace-header__crumb workspace-header__crumb--current">{{
          crumb.label
        }}</span>
        <span v-if="index < breadcrumbs.length - 1" class="workspace-header__crumb-sep">/</span>
      </template>
    </nav>

    <div class="workspace-header__row">
      <div class="workspace-header__identity">
        <div class="workspace-header__title-row">
          <h1 class="workspace-header__title">{{ title }}</h1>
          <BaseBadge v-if="status" :variant="status.variant ?? 'neutral'">{{
            status.label
          }}</BaseBadge>
        </div>
        <p v-if="subtitle" class="workspace-header__subtitle">{{ subtitle }}</p>
      </div>

      <div v-if="$slots.actions" class="workspace-header__actions">
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>

<style scoped>
.workspace-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.workspace-header__breadcrumbs {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.workspace-header__crumb {
  color: var(--color-text-muted);
  text-decoration: none;
}

.workspace-header__crumb:hover {
  color: var(--color-text-primary);
}

.workspace-header__crumb--current {
  color: var(--color-text-secondary);
}

.workspace-header__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.workspace-header__title-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.workspace-header__title {
  font-size: var(--font-size-2xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.workspace-header__subtitle {
  margin-top: var(--space-1);
  color: var(--color-text-secondary);
}

.workspace-header__actions {
  flex-shrink: 0;
}
</style>
