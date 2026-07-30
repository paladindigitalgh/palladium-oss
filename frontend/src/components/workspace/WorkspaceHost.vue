<script setup lang="ts">
/**
 * The one region of AppShell that changes when the route changes
 * (this milestone's goal 1: "Changing routes should only replace the
 * Workspace Host"). AppShell mounts this once and never unmounts it;
 * only the routed view inside it swaps.
 */
</script>

<template>
  <main class="workspace-host" id="main-content" tabindex="-1">
    <RouterView v-slot="{ Component, route }">
      <Transition name="workspace-fade" mode="out-in">
        <component :is="Component" :key="route.fullPath" />
      </Transition>
    </RouterView>
  </main>
</template>

<style scoped>
.workspace-host {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: var(--space-6);
}

.workspace-fade-enter-active,
.workspace-fade-leave-active {
  transition: opacity var(--motion-normal) var(--motion-ease);
}

.workspace-fade-enter-from,
.workspace-fade-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .workspace-fade-enter-active,
  .workspace-fade-leave-active {
    transition: none;
  }
}
</style>
