<script setup lang="ts">
/**
 * Skeleton only (explicit milestone scope): renders a static label/value
 * grid. It intentionally has no editing, validation, or field-type
 * awareness -- that belongs to a future PropertyEditor
 * (docs/11-COMPONENT-ARCHITECTURE.md's Forms section), which is out of
 * scope for this milestone (no CRUD forms).
 */
defineProps<{
  properties: { label: string; value: string }[]
}>()
</script>

<template>
  <dl class="base-property-grid">
    <div v-for="property in properties" :key="property.label" class="base-property-grid__row">
      <dt class="base-property-grid__label">{{ property.label }}</dt>
      <dd class="base-property-grid__value">{{ property.value }}</dd>
    </div>
  </dl>
</template>

<style scoped>
.base-property-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-4);
  margin: 0;
}

.base-property-grid__row {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.base-property-grid__label {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.base-property-grid__value {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}
</style>
