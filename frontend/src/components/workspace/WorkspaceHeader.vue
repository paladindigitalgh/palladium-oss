<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import BaseBadge from '@/components/base/BaseBadge.vue'

/**
 * The object-identity region of a Detail Workspace
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail Workspace Structure":
 * object identity, operational status, primary identifying information,
 * primary actions). Generic across every future domain -- it only knows
 * about a title, an optional status badge, an optional list of metadata
 * strings, and breadcrumbs, never about what kind of entity it is
 * heading. `metadata` is plain strings rather than a typed object for
 * exactly that reason: "Account #48213", "Joined 2023", and "16 PON
 * ports" are all just strings to this component.
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
  metadata?: string[]
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
        <ul v-if="metadata?.length" class="workspace-header__metadata">
          <li v-for="(item, index) in metadata" :key="index" class="workspace-header__metadata-item">
            {{ item }}
          </li>
        </ul>
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

.workspace-header__metadata {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1) var(--space-3);
  margin-top: var(--space-2);
}

.workspace-header__metadata-item {
  position: relative;
  padding-left: var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.workspace-header__metadata-item:first-child {
  padding-left: 0;
}

.workspace-header__metadata-item:not(:first-child)::before {
  content: '';
  position: absolute;
  left: 2px;
  top: 50%;
  width: 3px;
  height: 3px;
  border-radius: var(--radius-full);
  background-color: var(--color-text-muted);
  transform: translateY(-50%);
}

.workspace-header__actions {
  flex-shrink: 0;
}
</style>
