<script setup lang="ts">
/**
 * A plain title + description primitive, distinct from WorkspaceHeader
 * (components/workspace/WorkspaceHeader.vue): WorkspaceHeader is the
 * entity-identity region of a Workspace (status, breadcrumbs, primary
 * actions), while BasePageTitle is the generic heading used anywhere a
 * section needs a title without being a full Workspace.
 */
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="base-page-title">
    <div class="base-page-title__text">
      <h1 class="base-page-title__heading">{{ title }}</h1>
      <p v-if="description" class="base-page-title__description">{{ description }}</p>
    </div>
    <div v-if="$slots.default" class="base-page-title__actions">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.base-page-title {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.base-page-title__heading {
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.base-page-title__description {
  margin-top: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.base-page-title__actions {
  display: flex;
  gap: var(--space-2);
  flex-shrink: 0;
}
</style>
