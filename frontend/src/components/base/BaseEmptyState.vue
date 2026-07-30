<script setup lang="ts">
import BaseIcon, { type IconName } from './BaseIcon.vue'

/**
 * docs/08-DESIGN-SYSTEM.md section 20: every empty state should explain
 * why it's empty and offer a next action. `title` carries the "why",
 * `description` the "is this expected", and the default slot is reserved
 * for the action (typically a BaseButton) -- callers decide the action
 * since this component has no business logic of its own.
 */
withDefaults(
  defineProps<{
    icon?: IconName
    title: string
    description?: string
  }>(),
  { icon: 'inventory' },
)
</script>

<template>
  <div class="base-empty-state">
    <BaseIcon :name="icon" size="lg" class="base-empty-state__icon" />
    <p class="base-empty-state__title">{{ title }}</p>
    <p v-if="description" class="base-empty-state__description">{{ description }}</p>
    <div v-if="$slots.default" class="base-empty-state__action">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.base-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: var(--space-7) var(--space-5);
  gap: var(--space-2);
  color: var(--color-text-secondary);
}

.base-empty-state__icon {
  color: var(--color-text-muted);
  margin-bottom: var(--space-2);
}

.base-empty-state__title {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.base-empty-state__description {
  font-size: var(--font-size-sm);
  max-width: 32ch;
}

.base-empty-state__action {
  margin-top: var(--space-3);
}
</style>
