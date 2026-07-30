<script setup lang="ts">
import BaseIcon from './BaseIcon.vue'

/**
 * docs/08-DESIGN-SYSTEM.md section 20: every error should explain what
 * happened, its impact, and a corrective action. `title` covers "what
 * happened", `description` covers impact, and the default slot is for
 * the corrective action (e.g. a Retry BaseButton) -- left to the caller
 * since retry semantics differ per data source.
 */
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="base-error-state" role="alert">
    <BaseIcon name="alert" size="lg" class="base-error-state__icon" />
    <p class="base-error-state__title">{{ title }}</p>
    <p v-if="description" class="base-error-state__description">{{ description }}</p>
    <div v-if="$slots.default" class="base-error-state__action">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.base-error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: var(--space-7) var(--space-5);
  gap: var(--space-2);
}

.base-error-state__icon {
  color: var(--color-error);
  margin-bottom: var(--space-2);
}

.base-error-state__title {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.base-error-state__description {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  max-width: 40ch;
}

.base-error-state__action {
  margin-top: var(--space-3);
}
</style>
