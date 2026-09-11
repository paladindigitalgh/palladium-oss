<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import BaseIcon from '@/components/base/BaseIcon.vue'
import { NAV_ITEMS, type NavItem } from '@/router/navigation'
import { useSidebar } from '@/composables/useSidebar'
import { useTheme } from '@/composables/useTheme'

/**
 * docs' this-milestone goal 5: active state, icons, responsive behavior.
 *
 * "Collapsible sections" was originally not implemented -- NAV_ITEMS was
 * a flat list with no grouping defined anywhere in the navigation docs.
 * Administration is now the first item with `children` (navigation.ts),
 * at the user's explicit request: it renders as a toggle button that
 * never navigates itself, expanding in place to reveal its children as
 * indented links, rather than being its own page. Default collapsed,
 * but a child route being active forces it open too (isExpanded below)
 * -- landing on /administration/providers directly (a refresh, a
 * bookmark) should never hide which section you're in. This is
 * unrelated to `collapsed` (the whole sidebar's icon-only rail mode via
 * useSidebar) -- both concepts happen to use the word "collapsed" for
 * unrelated things, one per-item and one for the whole sidebar.
 */
const { collapsed, toggleCollapsed, mobileOpen, closeMobile, isMobileViewport } = useSidebar()
const { theme } = useTheme()

// The horizontal wordmark and the collapsed-rail symbol are each a
// single flat color, so neither can survive both a light and a dark
// sidebar on its own -- the "-dark" variants are the near-white-on-navy
// versions made specifically for dark mode.
const brandLogo = computed(() =>
  theme.value === 'dark' ? '/palladium-logo-horizontal-dark.png' : '/palladium-logo-horizontal.png',
)
const brandMark = computed(() =>
  theme.value === 'dark' ? '/palladium-logo-symbol-dark.png' : '/palladium-favicon-512.png',
)

const route = useRoute()
const expandedIds = ref<Set<string>>(new Set())

function isChildActive(item: NavItem): boolean {
  return item.children?.some((child) => route.path.startsWith(child.path)) ?? false
}

function isExpanded(item: NavItem): boolean {
  return expandedIds.value.has(item.id) || isChildActive(item)
}

function toggleExpanded(id: string) {
  const next = new Set(expandedIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  expandedIds.value = next
}

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
      <img v-if="collapsed" :src="brandMark" alt="Palladium" class="app-sidebar__brand-mark" />
      <img v-else :src="brandLogo" alt="Palladium" class="app-sidebar__brand-logo" />
    </div>

    <nav class="app-sidebar__nav" aria-label="Primary">
      <template v-for="item in NAV_ITEMS" :key="item.id">
        <RouterLink
          v-if="!item.children"
          :to="item.path"
          class="app-sidebar__link"
          active-class="app-sidebar__link--active"
          :title="collapsed ? item.label : undefined"
          @click="closeMobile"
        >
          <BaseIcon :name="item.icon" />
          <span v-if="!collapsed" class="app-sidebar__label">{{ item.label }}</span>
        </RouterLink>

        <template v-else>
          <button
            type="button"
            class="app-sidebar__link app-sidebar__link--toggle"
            :class="{ 'app-sidebar__link--active': isChildActive(item) }"
            :aria-expanded="isExpanded(item)"
            :title="collapsed ? item.label : undefined"
            @click="toggleExpanded(item.id)"
          >
            <BaseIcon :name="item.icon" />
            <span v-if="!collapsed" class="app-sidebar__label">{{ item.label }}</span>
            <BaseIcon
              v-if="!collapsed"
              name="chevron-down"
              size="sm"
              class="app-sidebar__chevron"
              :class="{ 'app-sidebar__chevron--collapsed': !isExpanded(item) }"
            />
          </button>

          <div v-if="!collapsed && isExpanded(item)" class="app-sidebar__children">
            <RouterLink
              v-for="child in item.children"
              :key="child.id"
              :to="child.path"
              class="app-sidebar__link app-sidebar__link--child"
              active-class="app-sidebar__link--active"
              @click="closeMobile"
            >
              <span class="app-sidebar__label">{{ child.label }}</span>
            </RouterLink>
          </div>
        </template>
      </template>
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

.app-sidebar--collapsed .app-sidebar__brand {
  padding: 0 var(--space-2);
  justify-content: center;
}

.app-sidebar__brand-mark {
  width: 36px;
  height: 36px;
  object-fit: contain;
  flex-shrink: 0;
}

.app-sidebar__brand-logo {
  height: 40px;
  width: auto;
  max-width: 100%;
  object-fit: contain;
}

.app-sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3);
  overflow-y: auto;
  /* Labels unmount instantly on collapse (v-if) but the sidebar's width
     keeps animating for --motion-normal, so on expand the full-width
     text is briefly wider than the still-narrow container -- without
     this, that transient overflow triggers a horizontal scrollbar. */
  overflow-x: hidden;
  flex: 1;
}

.app-sidebar__link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  text-decoration: none;
  font: inherit;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  text-align: left;
  cursor: pointer;
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

.app-sidebar__chevron {
  margin-left: auto;
  flex-shrink: 0;
  transition: transform var(--motion-normal) var(--motion-ease);
}

.app-sidebar__chevron--collapsed {
  transform: rotate(-90deg);
}

.app-sidebar__children {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: var(--space-1) 0;
}

.app-sidebar__link--child {
  /* Aligns the child's label under the parent's label, not its icon:
     parent padding-left + icon width + the gap between icon and label. */
  padding-left: calc(var(--space-3) + var(--icon-size-md) + var(--space-3));
}

@media (prefers-reduced-motion: reduce) {
  .app-sidebar__chevron {
    transition: none;
  }
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
