<script setup lang="ts">
/**
 * A labeled native checkbox, the base primitive for a boolean filter
 * toggle (docs/08-DESIGN-SYSTEM.md section 13, "Forms & Inputs") --
 * mirrors BaseSelect.vue's shape (a native control wrapped in a
 * `<label>`, styled via the same design tokens) for the cases a
 * dropdown does not fit, e.g. "Include Archived" beside a Collection
 * View's status filter.
 */
withDefaults(
  defineProps<{
    label: string
  }>(),
  {},
)

const model = defineModel<boolean>({ required: true })
</script>

<template>
  <label class="base-checkbox">
    <input v-model="model" type="checkbox" class="base-checkbox__input" />
    <span class="base-checkbox__label">{{ label }}</span>
  </label>
</template>

<style scoped>
.base-checkbox {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
}

.base-checkbox__input {
  width: 16px;
  height: 16px;
  accent-color: var(--color-brand);
  cursor: pointer;
}

.base-checkbox__label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  white-space: nowrap;
}
</style>
