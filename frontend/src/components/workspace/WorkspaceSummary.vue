<script setup lang="ts">
import BaseCard from '@/components/base/BaseCard.vue'

/**
 * The "high-value information first" region (docs/11-COMPONENT-
 * ARCHITECTURE.md, "Workspace Summary": customer contact details, ONU
 * serial number, OLT model, service package -- all domain-specific, so
 * this component stays a plain elevated container. Callers compose the
 * actual summary fields inside (typically a BasePropertyGrid) rather
 * than this component knowing about any entity's shape.
 */
defineProps<{ title?: string }>()
</script>

<template>
  <BaseCard elevated class="workspace-summary">
    <h2 v-if="title" class="workspace-summary__title">{{ title }}</h2>
    <slot />
  </BaseCard>
</template>

<style scoped>
.workspace-summary__title {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  margin-bottom: var(--space-4);
}
</style>
