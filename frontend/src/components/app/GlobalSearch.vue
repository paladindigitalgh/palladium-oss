<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * Placeholder only (out of scope: "Do NOT implement real search" /
 * "No mock provisioning logic"). It renders the search affordance and
 * the "/" focus shortcut docs/04-NAVIGATION.md section 10 recommends,
 * but submitting does nothing yet -- there is no search backend for it
 * to call.
 */
const inputRef = ref<HTMLInputElement | null>(null)

function isTypingInField(target: EventTarget | null): boolean {
  const element = target as HTMLElement | null
  return element?.tagName === 'INPUT' || element?.tagName === 'TEXTAREA' || element?.isContentEditable === true
}

function handleShortcut(event: KeyboardEvent) {
  if (event.key === '/' && !isTypingInField(event.target)) {
    event.preventDefault()
    inputRef.value?.focus()
  }
}

onMounted(() => window.addEventListener('keydown', handleShortcut))
onUnmounted(() => window.removeEventListener('keydown', handleShortcut))
</script>

<template>
  <form class="global-search" role="search" @submit.prevent>
    <BaseIcon name="search" size="sm" class="global-search__icon" />
    <input
      ref="inputRef"
      type="search"
      class="global-search__input"
      placeholder="Search customers, services, devices..."
      aria-label="Global search"
    />
    <kbd class="global-search__shortcut">/</kbd>
  </form>
</template>

<style scoped>
.global-search {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex: 1;
  max-width: 480px;
  padding: 0 var(--space-3);
  height: 36px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background-color: var(--color-bg);
  color: var(--color-text-muted);
}

.global-search:focus-within {
  border-color: var(--color-brand);
}

.global-search__icon {
  flex-shrink: 0;
}

.global-search__input {
  flex: 1;
  border: none;
  background: transparent;
  outline: none;
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  min-width: 0;
}

.global-search__input::placeholder {
  color: var(--color-text-muted);
}

.global-search__shortcut {
  flex-shrink: 0;
  padding: 1px var(--space-2);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-strong);
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
