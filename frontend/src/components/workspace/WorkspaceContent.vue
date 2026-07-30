<script setup lang="ts">
/**
 * The primary-content landmark region (docs/11-COMPONENT-ARCHITECTURE.md:
 * "Workspace Content: the primary domain-specific information"). Unlike
 * WorkspaceSummary it is not itself a card -- workspaces typically stack
 * several BaseSection cards inside it, so this component only provides
 * the region's spacing rhythm, not its own visual container.
 */
defineProps<{ title?: string }>()
</script>

<template>
  <section class="workspace-content">
    <h2 v-if="title" class="workspace-content__title">{{ title }}</h2>
    <slot />
  </section>
</template>

<style scoped>
.workspace-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.workspace-content__title {
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}
</style>
