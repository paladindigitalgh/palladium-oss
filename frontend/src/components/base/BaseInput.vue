<script setup lang="ts">
/**
 * A labeled text input, styled identically to BaseSelect.vue (same
 * label + control shape and tokens) -- the base primitive every create
 * form in components/dialogs/ builds on, extracted once a second form
 * needed the same input markup LoginView.vue first wrote inline.
 */
withDefaults(
  defineProps<{
    label: string
    type?: 'text' | 'email'
    required?: boolean
  }>(),
  { type: 'text', required: false },
)

const model = defineModel<string>({ required: true })
</script>

<template>
  <label class="base-input">
    <span class="base-input__label">{{ label }}</span>
    <input v-model="model" :type="type" :required="required" class="base-input__control" />
  </label>
</template>

<style scoped>
.base-input {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.base-input__label {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.base-input__control {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background-color: var(--color-surface);
  color: var(--color-text-primary);
  font: inherit;
  font-size: var(--font-size-sm);
}

.base-input__control:hover {
  border-color: var(--color-text-muted);
}

.base-input__control:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
}
</style>
