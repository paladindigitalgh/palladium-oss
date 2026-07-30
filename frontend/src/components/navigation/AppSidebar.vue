<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { RouterLink } from 'vue-router'
import BaseIcon from '@/components/base/BaseIcon.vue'
import { NAV_ITEMS } from '@/router/navigation'
import { useSidebar } from '@/composables/useSidebar'

/**
 * docs' this-milestone goal 5: active state, icons, responsive behavior.
 * "Collapsible sections" is intentionally not implemented -- NAV_ITEMS
 * is a flat, eight-item list with no grouping defined anywhere in the
 * navigation docs, so there is no section boundary to collapse yet. What
 * *is* collapsible is the sidebar itself (icon-only rail via
 * useSidebar), which is the responsive/space-saving behavior goal 5
 * actually asks for.
 */
const { collapsed, toggleCollapsed, mobileOpen, closeMobile, isMobileViewport } = useSidebar()

// While off-canvas (mobile viewport and not open), the sidebar must not
// be part of the tab order or hit-testable -- otherwise a keyboard user
// can tab into links that are invisible off-screen. `inert` removes the
// whole subtree from focus, hit-testing, and assistive tech in one
// attribute, and only applies when the sidebar is actually off-screen
// (desktop never sets isMobileViewport, so it never applies there).
const hidden = () => isMobileViewport.value && !mobileOpen.value

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && mobileOpen.value) {
    closeMobile()
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <aside
    class="app-sidebar"
    :class="{ 'app-sidebar--collapsed': collapsed, 'app-sidebar--mobile-open': mobileOpen }"
    :inert="hidden()"
  >
    <div class="app-sidebar__brand">
      <span class="app-sidebar__brand-mark" aria-hidden="true">P</span>
      <span v-if="!collapsed" class="app-sidebar__brand-name">Palladium</span>
    </div>

    <nav class="app-sidebar__nav" aria-label="Primary">
      <RouterLink
        v-for="item in NAV_ITEMS"
        :key="item.id"
        :to="item.path"
        class="app-sidebar__link"
        active-class="app-sidebar__link--active"
        :title="collapsed ? item.label : undefined"
        @click="closeMobile"
      >
        <BaseIcon :name="item.icon" />
        <span v-if="!collapsed" class="app-sidebar__label">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <button
      type="button"
      class="app-sidebar__collapse-toggle"
      :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
      @click="toggleCollapsed"
    >
      <BaseIcon :name="collapsed ? 'chevron-right' : 'chevron-left'" size="sm" />
      <span v-if="!collapsed">Collapse</span>
    </button>
  </aside>

  <div
    v-if="mobileOpen"
    class="app-sidebar__scrim"
    aria-hidden="true"
    @click="closeMobile"
  />
</template>

<style scoped>
.app-sidebar {
  display: flex;
  flex-direction: column;
  width: var(--shell-sidebar-width);
  flex-shrink: 0;
  background-color: var(--color-surface);
  border-right: 1px solid var(--color-border);
  transition: width var(--motion-normal) var(--motion-ease);
}

.app-sidebar--collapsed {
  width: var(--shell-sidebar-width-collapsed);
}

.app-sidebar__brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: var(--shell-topnav-height);
  padding: 0 var(--space-4);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.app-sidebar__brand-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  background-color: var(--color-brand);
  color: var(--color-brand-contrast);
  font-weight: var(--font-weight-semibold);
  flex-shrink: 0;
}

.app-sidebar__brand-name {
  font-weight: var(--font-weight-semibold);
  white-space: nowrap;
}

.app-sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3);
  overflow-y: auto;
  flex: 1;
}

.app-sidebar__link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.app-sidebar__link:hover {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.app-sidebar__link--active {
  background-color: var(--color-info-bg);
  color: var(--color-brand);
}

.app-sidebar__label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.app-sidebar__collapse-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: var(--font-size-sm);
}

.app-sidebar__collapse-toggle:hover {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.app-sidebar__scrim {
  display: none;
}

@media (max-width: 960px) {
  .app-sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    z-index: 40;
    width: var(--shell-sidebar-width);
    transform: translateX(-100%);
    transition: transform var(--motion-normal) var(--motion-ease);
    box-shadow: var(--shadow-lg);
  }

  .app-sidebar--mobile-open {
    transform: translateX(0);
  }

  .app-sidebar--collapsed {
    width: var(--shell-sidebar-width);
  }

  .app-sidebar__scrim {
    display: block;
    position: fixed;
    inset: 0;
    background-color: rgba(10, 13, 20, 0.4);
    z-index: 30;
  }
}
</style>
