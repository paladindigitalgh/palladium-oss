<script setup lang="ts">
import BaseIcon from './BaseIcon.vue'

/**
 * The generic dialog shell every create form and confirmation dialog
 * builds on (docs/07-UI-ARCHITECTURE.md section 14, "Dialogs & Drawers").
 * Controlled by the parent via `open` rather than owning its own state
 * (compare composables/useDisclosure.ts, used by small popovers that own
 * their own open/close) -- a dialog's open state is a decision the
 * caller makes (a button click, a form's success), not something this
 * shell should track independently.
 *
 * Teleported to <body> so it stacks above the Detail Workspace's sticky
 * Contents navigation and AppShell regardless of where it is mounted.
 */
defineProps<{
  open: boolean
  title: string
}>()

const emit = defineEmits<{ (event: 'close'): void }>()

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="base-modal-overlay" @click.self="emit('close')" @keydown="onKeydown">
      <div class="base-modal" role="dialog" aria-modal="true" :aria-labelledby="`${title}-heading`">
        <div class="base-modal__header">
          <h2 :id="`${title}-heading`" class="base-modal__title">{{ title }}</h2>
          <button type="button" class="base-modal__close" aria-label="Close" @click="emit('close')">
            <BaseIcon name="close" size="sm" />
          </button>
        </div>
        <div class="base-modal__body">
          <slot />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.base-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgb(0 0 0 / 0.4);
  padding: var(--space-4);
}

.base-modal {
  width: 100%;
  max-width: 480px;
  max-height: calc(100vh - var(--space-8));
  overflow-y: auto;
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
}

.base-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
}

.base-modal__title {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.base-modal__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  flex-shrink: 0;
}

.base-modal__close:hover {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.base-modal__body {
  padding: var(--space-5);
}
</style>
