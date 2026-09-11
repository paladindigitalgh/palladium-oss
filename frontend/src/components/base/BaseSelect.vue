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
withDefaults(
  defineProps<{
    label: string
    options: { value: string; label: string }[]
    /**
     * Visually hides the label (e.g. a per-row control in a table whose
     * column header already names the field) while keeping it in the DOM
     * for assistive technology -- never omit the label prop itself just
     * because a caller sets this.
     */
    hideLabel?: boolean
  }>(),
  { hideLabel: false },
)

const model = defineModel<string>({ required: true })
</script>

<template>
  <label class="base-select">
    <span class="base-select__label" :class="{ 'base-select__label--hidden': hideLabel }">{{ label }}</span>
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
  /* Without this, .base-select__label--hidden below (position: absolute,
     no positioned ancestor of its own) resolves its containing block all
     the way up to the initial containing block instead of this element,
     silently making the whole page scrollable past its real content --
     the exact bug BaseButton.vue's own disabled-reason span had, fixed
     there in commit 98e09a4 by giving .base-button position: relative;
     same fix, same reasoning, one component over. */
  position: relative;
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

.base-select__label--hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
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
