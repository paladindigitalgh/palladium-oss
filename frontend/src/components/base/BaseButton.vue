<script setup lang="ts">
/**
 * docs/08-DESIGN-SYSTEM.md section 12: one clear primary action per
 * view, secondary actions visually subordinate, destructive actions
 * clearly distinguished, and disabled actions should explain why --
 * hence `disabledReason`, surfaced as a title tooltip rather than a new
 * tooltip component this early.
 */
withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'destructive' | 'ghost'
    size?: 'sm' | 'md'
    type?: 'button' | 'submit'
    disabled?: boolean
    disabledReason?: string
  }>(),
  { variant: 'secondary', size: 'md', type: 'button', disabled: false },
)
</script>

<template>
  <button
    class="base-button"
    :class="[`base-button--${variant}`, `base-button--${size}`]"
    :type="type"
    :disabled="disabled"
    :title="disabled ? disabledReason : undefined"
  >
    <slot />
  </button>
</template>

<style scoped>
.base-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition:
    background-color var(--motion-fast) var(--motion-ease),
    border-color var(--motion-fast) var(--motion-ease),
    opacity var(--motion-fast) var(--motion-ease);
  white-space: nowrap;
}

.base-button--md {
  padding: var(--space-2) var(--space-4);
  font-size: var(--font-size-md);
}

.base-button--sm {
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-sm);
}

.base-button--primary {
  background-color: var(--color-brand);
  color: var(--color-brand-contrast);
}

.base-button--primary:hover:not(:disabled) {
  background-color: var(--color-brand-strong);
}

.base-button--secondary {
  background-color: var(--color-surface);
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
}

.base-button--secondary:hover:not(:disabled) {
  background-color: var(--color-bg);
}

.base-button--destructive {
  background-color: var(--color-error-bg);
  color: var(--color-error);
  border-color: var(--color-error-bg);
}

.base-button--destructive:hover:not(:disabled) {
  border-color: var(--color-error);
}

.base-button--ghost {
  background-color: transparent;
  color: var(--color-text-secondary);
}

.base-button--ghost:hover:not(:disabled) {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.base-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
