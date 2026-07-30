<script setup lang="ts">
/**
 * The Workspace action bar. Purely a layout container -- it renders
 * whatever BaseButtons the composing workspace passes in and takes no
 * position on what those actions do (docs/11-COMPONENT-ARCHITECTURE.md:
 * "business logic must not live inside presentation components"). The
 * `secondary` slot renders first and visually subordinate, `primary`
 * last and prominent, per docs/08-DESIGN-SYSTEM.md section 12.
 */
</script>

<template>
  <div class="workspace-actions">
    <div v-if="$slots.secondary" class="workspace-actions__secondary">
      <slot name="secondary" />
    </div>
    <div v-if="$slots.primary" class="workspace-actions__primary">
      <slot name="primary" />
    </div>
  </div>
</template>

<style scoped>
.workspace-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.workspace-actions__secondary,
.workspace-actions__primary {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
</style>
