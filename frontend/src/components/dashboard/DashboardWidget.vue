<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseIcon, { type IconName } from '@/components/base/BaseIcon.vue'

/**
 * The shared header shell for every Dashboard widget (Milestone 2's
 * Widget Design: title, small icon, "View All ->" action, consistent
 * spacing/typography across all four widgets). Lives in
 * components/dashboard/, not components/data-display/ or base/: no
 * other workspace has asked for this exact "titled card with a View All
 * link" shape yet, and per docs/11-COMPONENT-ARCHITECTURE.md's Future
 * Evolution note, a pattern is promoted into the shared library once a
 * second place actually needs it, not speculatively. If a future
 * workspace wants the same shape, that is the signal to promote it.
 */
defineProps<{
  title: string
  icon: IconName
  viewAllTo: RouteLocationRaw
}>()
</script>

<template>
  <BaseCard class="dashboard-widget">
    <div class="dashboard-widget__header">
      <div class="dashboard-widget__heading">
        <BaseIcon :name="icon" size="sm" class="dashboard-widget__icon" />
        <h2 class="dashboard-widget__title">{{ title }}</h2>
      </div>
      <RouterLink :to="viewAllTo" class="dashboard-widget__view-all">
        View All
        <BaseIcon name="arrow-right" size="sm" />
      </RouterLink>
    </div>
    <div class="dashboard-widget__body">
      <slot />
    </div>
  </BaseCard>
</template>

<style scoped>
.dashboard-widget {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.dashboard-widget__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.dashboard-widget__heading {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.dashboard-widget__icon {
  color: var(--color-text-secondary);
}

.dashboard-widget__title {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dashboard-widget__view-all {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex-shrink: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
  text-decoration: none;
  border-radius: var(--radius-sm);
}

.dashboard-widget__view-all:hover {
  color: var(--color-brand);
}

.dashboard-widget__body {
  flex: 1;
}
</style>
