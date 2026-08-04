<script setup lang="ts">
import BaseIcon from './BaseIcon.vue'

/**
 * A labeled native <select>, the base primitive behind every filter
 * control (docs/08-DESIGN-SYSTEM.md section 13, "Forms & Inputs"). Native
 * <select> is used rather than a custom listbox: it is keyboard and
 * screen-reader accessible for free, and filtering does not need the
 * multi-select or async-search behavior that would justify a bespoke
 * widget (docs/09-WORKSPACE-SPECIFICATIONS.md, "Do not overcomplicate
 * filtering").
 */
defineProps<{
  label: string
  options: { value: string; label: string }[]
}>()

const model = defineModel<string>({ required: true })
</script>

<template>
  <label class="base-select">
    <span class="base-select__label">{{ label }}</span>
    <span class="base-select__control">
      <select v-model="model" class="base-select__input">
        <option v-for="option in options" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
      <BaseIcon name="chevron-down" size="sm" class="base-select__chevron" />
    </span>
  </label>
</template>

<style scoped>
.base-select {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.base-select__label {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.base-select__control {
  position: relative;
  display: flex;
  align-items: center;
}

.base-select__input {
  width: 100%;
  appearance: none;
  padding: var(--space-2) var(--space-7) var(--space-2) var(--space-3);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background-color: var(--color-surface);
  color: var(--color-text-primary);
  font: inherit;
  font-size: var(--font-size-sm);
  cursor: pointer;
}

.base-select__input:hover {
  border-color: var(--color-text-muted);
}

.base-select__chevron {
  position: absolute;
  right: var(--space-3);
  color: var(--color-text-muted);
  pointer-events: none;
}
</style>
