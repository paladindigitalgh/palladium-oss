<script setup lang="ts">
import { useTemplateRef } from 'vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import { useDisclosure } from '@/composables/useDisclosure'

/**
 * Placeholder only (out of scope: "Do NOT implement notifications").
 * Structurally real -- a working toggle, popover, and empty state -- but
 * with no notification source wired in, since none exists yet.
 */
const root = useTemplateRef<HTMLElement>('root')
const { open, toggle } = useDisclosure(root)
</script>

<template>
  <div ref="root" class="notification-center">
    <button
      type="button"
      class="notification-center__trigger"
      aria-label="Notifications"
      :aria-expanded="open"
      @click="toggle"
    >
      <BaseIcon name="bell" />
    </button>

    <div v-if="open" class="notification-center__panel" role="menu">
      <p class="notification-center__title">Notifications</p>
      <BaseEmptyState
        icon="bell"
        title="No notifications yet"
        description="You're all caught up. New notifications will appear here."
      />
    </div>
  </div>
</template>

<style scoped>
.notification-center {
  position: relative;
}

.notification-center__trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.notification-center__trigger:hover {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.notification-center__panel {
  position: absolute;
  top: calc(100% + var(--space-2));
  right: 0;
  width: 300px;
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: var(--space-3);
  z-index: 50;
}

.notification-center__title {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  padding: var(--space-1) var(--space-2);
}
</style>
