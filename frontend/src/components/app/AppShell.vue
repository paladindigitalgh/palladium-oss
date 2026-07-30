<script setup lang="ts">
import AppSidebar from '@/components/navigation/AppSidebar.vue'
import TopNavigation from '@/components/navigation/TopNavigation.vue'
import WorkspaceHost from '@/components/workspace/WorkspaceHost.vue'

/**
 * This milestone's goal 1: "The AppShell should remain mounted for the
 * lifetime of the application... Changing routes should only replace the
 * Workspace Host." App.vue mounts AppShell exactly once; every route
 * change re-renders only the <RouterView> inside WorkspaceHost.
 */
</script>

<template>
  <div class="app-shell">
    <a href="#main-content" class="app-shell__skip-link">Skip to main content</a>
    <AppSidebar />
    <div class="app-shell__main">
      <TopNavigation />
      <WorkspaceHost />
    </div>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

/*
 * WCAG 2.4.1 "Bypass Blocks": the sidebar alone is 8 tab stops before
 * reaching page content, repeated on every navigation. Hidden until
 * focused, per the standard skip-link pattern.
 */
.app-shell__skip-link {
  position: fixed;
  top: -100%;
  left: var(--space-4);
  z-index: 100;
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-sm);
  background-color: var(--color-brand);
  color: var(--color-brand-contrast);
  font-weight: var(--font-weight-medium);
  text-decoration: none;
  transition: top var(--motion-fast) var(--motion-ease);
}

.app-shell__skip-link:focus-visible {
  top: var(--space-4);
}

.app-shell__main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
</style>
