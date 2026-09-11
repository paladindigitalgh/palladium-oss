<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import BaseIcon from '@/components/base/BaseIcon.vue'
import Breadcrumbs from './Breadcrumbs.vue'
import GlobalSearch from '@/components/app/GlobalSearch.vue'
import NotificationCenter from '@/components/app/NotificationCenter.vue'
import UserMenu from '@/components/app/UserMenu.vue'
import { useSidebar } from '@/composables/useSidebar'
import { useBreadcrumb } from '@/composables/useBreadcrumb'

/**
 * This milestone's goal 6: application title/logo (via the shell-level
 * breadcrumb root), breadcrumb area, global search, notifications, and
 * user menu. Composes GlobalSearch/NotificationCenter/UserMenu rather
 * than AppShell rendering them as flat siblings -- see GlobalSearch's
 * placement note in the milestone summary for why.
 *
 * The breadcrumb is a directory-style trail (2026-09-11, user's explicit
 * request), not a shallow "which section" indicator -- see
 * useBreadcrumb.ts's own doc comment for how it's populated. The back
 * arrow navigates to the second-to-last item's `to`, the same "up one
 * level" a real directory back button does; it disappears whenever
 * there isn't one (a root page with only one segment, or a segment
 * whose predecessor has no real page to link to, e.g. Services or a
 * bare Administration crumb) rather than rendering disabled.
 */
const router = useRouter()
const { mobileOpen, toggleMobile } = useSidebar()
const { items: breadcrumbItems } = useBreadcrumb()

const backTarget = computed(() => {
  const items = breadcrumbItems.value
  const parent = items[items.length - 2]
  return parent?.to ?? null
})

function goBack() {
  if (backTarget.value) router.push(backTarget.value)
}
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

    <button
      v-if="backTarget"
      type="button"
      class="top-navigation__back"
      aria-label="Back a level"
      @click="goBack"
    >
      <BaseIcon name="chevron-left" />
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

.top-navigation__back {
  display: flex;
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

.top-navigation__back:hover {
  background-color: var(--color-bg);
  color: var(--color-text-primary);
}

.top-navigation__breadcrumbs {
  flex-shrink: 0;
}

.top-navigation__search {
  margin-left: auto;
}

.top-navigation__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

@media (max-width: 960px) {
  .top-navigation__back {
    display: none;
  }

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
