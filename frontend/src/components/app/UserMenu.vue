<script setup lang="ts">
import { useTemplateRef } from 'vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { useDisclosure } from '@/composables/useDisclosure'
import { useTheme } from '@/composables/useTheme'

/**
 * Placeholder only ("No authentication work is required" -- goal 6).
 * There is no signed-in operator to display yet, so this shows a
 * generic account glyph rather than inventing a fake name or email.
 * The theme toggle lives here because Theme is "Global Layer" state
 * (docs/04-NAVIGATION.md section 3) conventionally reached through the
 * user/profile menu, and it is genuinely functional today. Profile and
 * Sign out are visibly disabled with a reason rather than being silent
 * dead buttons, since there is no identity or session to act on yet.
 */
const root = useTemplateRef<HTMLElement>('root')
const { open, toggle } = useDisclosure(root)
const { theme, toggleTheme } = useTheme()
</script>

<template>
  <div ref="root" class="user-menu">
    <button
      type="button"
      class="user-menu__trigger"
      aria-label="Account menu"
      :aria-expanded="open"
      @click="toggle"
    >
      <span class="user-menu__avatar"><BaseIcon name="user" size="sm" /></span>
      <BaseIcon name="chevron-down" size="sm" />
    </button>

    <div v-if="open" class="user-menu__panel" role="menu">
      <button type="button" class="user-menu__item" role="menuitem" @click="toggleTheme">
        <BaseIcon :name="theme === 'dark' ? 'sun' : 'moon'" size="sm" />
        <span>Switch to {{ theme === 'dark' ? 'light' : 'dark' }} theme</span>
      </button>

      <div class="user-menu__divider" />

      <BaseButton
        variant="ghost"
        size="sm"
        class="user-menu__item"
        disabled
        disabled-reason="Authentication is not implemented in this milestone"
      >
        Profile
      </BaseButton>
      <BaseButton
        variant="ghost"
        size="sm"
        class="user-menu__item"
        disabled
        disabled-reason="Authentication is not implemented in this milestone"
      >
        Sign out
      </BaseButton>
    </div>
  </div>
</template>

<style scoped>
.user-menu {
  position: relative;
}

.user-menu__trigger {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  height: 36px;
  padding: 0 var(--space-2);
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.user-menu__trigger:hover {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.user-menu__avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: var(--radius-full);
  background-color: var(--color-info-bg);
  color: var(--color-brand);
}

.user-menu__panel {
  position: absolute;
  top: calc(100% + var(--space-2));
  right: 0;
  width: 240px;
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: var(--space-2);
  z-index: 50;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.user-menu__item {
  width: 100%;
  justify-content: flex-start;
  text-align: left;
}

button.user-menu__item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-2);
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: var(--font-size-sm);
}

button.user-menu__item:hover {
  background-color: var(--color-bg);
}

.user-menu__divider {
  height: 1px;
  background-color: var(--color-border);
  margin: var(--space-1) 0;
}
</style>
