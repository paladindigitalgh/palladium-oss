<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import BaseIcon from '@/components/base/BaseIcon.vue'
import Breadcrumbs from './Breadcrumbs.vue'
import GlobalSearch from '@/components/app/GlobalSearch.vue'
import NotificationCenter from '@/components/app/NotificationCenter.vue'
import UserMenu from '@/components/app/UserMenu.vue'
import { useSidebar } from '@/composables/useSidebar'

/**
 * This milestone's goal 6: application title/logo (via the shell-level
 * breadcrumb root), breadcrumb area, global search, notifications, and
 * user menu. Composes GlobalSearch/NotificationCenter/UserMenu rather
 * than AppShell rendering them as flat siblings -- see GlobalSearch's
 * placement note in the milestone summary for why.
 */
const route = useRoute()
const { mobileOpen, toggleMobile } = useSidebar()

const breadcrumbItems = computed(() => [
  { label: 'Palladium', to: '/dashboard' },
  { label: typeof route.meta.title === 'string' ? route.meta.title : String(route.name ?? '') },
])
</script>

<template>
  <header class="top-navigation">
    <button
      type="button"
      class="top-navigation__menu-toggle"
      :aria-label="mobileOpen ? 'Close navigation' : 'Open navigation'"
      :aria-expanded="mobileOpen"
      @click="toggleMobile"
    >
      <BaseIcon :name="mobileOpen ? 'close' : 'menu'" />
    </button>

    <Breadcrumbs :items="breadcrumbItems" class="top-navigation__breadcrumbs" />

    <GlobalSearch class="top-navigation__search" />

    <div class="top-navigation__actions">
      <NotificationCenter />
      <UserMenu />
    </div>
  </header>
</template>

<style scoped>
.top-navigation {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  height: var(--shell-topnav-height);
  padding: 0 var(--space-5);
  border-bottom: 1px solid var(--color-border);
  background-color: var(--color-surface);
  flex-shrink: 0;
}

.top-navigation__menu-toggle {
  display: none;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  flex-shrink: 0;
}

.top-navigation__breadcrumbs {
  flex-shrink: 0;
}

.top-navigation__search {
  margin-left: var(--space-4);
}

.top-navigation__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
  flex-shrink: 0;
}

@media (max-width: 960px) {
  .top-navigation__menu-toggle {
    display: flex;
  }

  .top-navigation__breadcrumbs {
    display: none;
  }

  .top-navigation__search {
    margin-left: 0;
  }
}
</style>
