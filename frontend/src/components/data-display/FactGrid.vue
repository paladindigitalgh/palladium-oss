<script setup lang="ts">
import BaseIcon, { type IconName } from '@/components/base/BaseIcon.vue'

/**
 * A small grid of tinted, icon-labeled fact tiles -- the alternative to
 * BasePropertyGrid's plain label/value table when a section wants
 * grouped, fast-scanning facts instead of a wall of rows
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, Customer/Device Workspace
 * Summary sections: "Avoid giant property grids"). Extracted once
 * Customer Detail's Summary and Device Detail's Summary/Network/
 * Status/Configuration all needed the identical tile shape
 * (docs/11-COMPONENT-ARCHITECTURE.md, "Future Evolution": promote a
 * pattern once a second place needs it).
 */
export interface Fact {
  icon?: IconName
  label: string
  value: string
}

defineProps<{
  facts: Fact[]
}>()
</script>

<template>
  <div class="fact-grid">
    <div v-for="fact in facts" :key="fact.label" class="fact-grid__item">
      <BaseIcon v-if="fact.icon" :name="fact.icon" size="sm" class="fact-grid__icon" />
      <div class="fact-grid__text">
        <span class="fact-grid__label">{{ fact.label }}</span>
        <span class="fact-grid__value">{{ fact.value }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fact-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-3);
}

.fact-grid__item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background-color: var(--color-bg);
}

.fact-grid__icon {
  color: var(--color-text-muted);
  margin-top: 2px;
}

.fact-grid__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.fact-grid__label {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.fact-grid__value {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}
</style>
