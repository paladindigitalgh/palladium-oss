<script setup lang="ts">
/**
 * docs/08-DESIGN-SYSTEM.md section 16: "Badge appearance should be
 * driven entirely by semantic design tokens." Section 10 also requires
 * status to never depend on color alone, so BaseBadge always renders its
 * slot as text -- there is no icon-only / color-only mode.
 */
withDefaults(
  defineProps<{
    variant?: 'success' | 'warning' | 'error' | 'info' | 'neutral'
  }>(),
  { variant: 'neutral' },
)
</script>

<template>
  <span class="base-badge" :class="`base-badge--${variant}`">
    <slot />
  </span>
</template>

<style scoped>
.base-badge {
  display: inline-flex;
  align-items: center;
  padding: 0 var(--space-2);
  height: 22px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  line-height: 1;
}

.base-badge--neutral {
  background-color: var(--color-bg);
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.base-badge--success {
  background-color: var(--color-success-bg);
  color: var(--color-success);
}

.base-badge--warning {
  background-color: var(--color-warning-bg);
  color: var(--color-warning);
}

.base-badge--error {
  background-color: var(--color-error-bg);
  color: var(--color-error);
}

.base-badge--info {
  background-color: var(--color-info-bg);
  color: var(--color-info);
}
</style>
